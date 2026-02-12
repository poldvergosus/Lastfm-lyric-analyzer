package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"lastfm-lyrics/cache"
	"lastfm-lyrics/config"
	"lastfm-lyrics/models"
	"lastfm-lyrics/services"
)

type Handler struct {
	cfg     *config.Config
	cache   *cache.LyricsCache
	tasks   map[string]*models.TaskStatus
	tasksMu sync.RWMutex
}

func New(cfg *config.Config, c *cache.LyricsCache) *Handler {
	return &Handler{
		cfg:   cfg,
		cache: c,
		tasks: make(map[string]*models.TaskStatus),
	}
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}

	var req models.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	if req.Username == "" || req.From == "" || req.To == "" {
		writeJSON(w, 400, map[string]string{"error": "username, from, to are required"})
		return
	}

	if req.MaxTracks == 0 {
		req.MaxTracks = 500
	}
	req.ExcludeStopWords = true

	taskID := fmt.Sprintf("%x", md5.Sum(
		[]byte(req.Username+"_"+req.From+"_"+req.To),
	))

	h.tasksMu.RLock()
	existing, exists := h.tasks[taskID]
	h.tasksMu.RUnlock()

	if exists {
		switch existing.Phase {
		case "tracks", "lyrics", "analyzing":
			writeJSON(w, 200, map[string]string{
				"task_id": taskID,
				"status":  "already_running",
			})
			return
		}
	}

	h.tasksMu.Lock()
	h.tasks[taskID] = &models.TaskStatus{ID: taskID, Phase: "pending"}
	h.tasksMu.Unlock()

	go h.runAnalysis(taskID, req)
	writeJSON(w, 200, map[string]string{"task_id": taskID})
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {

	taskID := extractLastSegment(r.URL.Path)

	if taskID == "" {
		writeJSON(w, 400, map[string]string{"error": "task id required"})
		return
	}

	h.tasksMu.RLock()
	status, exists := h.tasks[taskID]
	h.tasksMu.RUnlock()

	if !exists {
		writeJSON(w, 404, map[string]string{"error": "task not found"})
		return
	}

	writeJSON(w, 200, status)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	total, found := h.cache.Stats()
	writeJSON(w, 200, map[string]interface{}{
		"status":      "ok",
		"cache_total": total,
		"cache_found": found,
	})
}

func (h *Handler) runAnalysis(taskID string, req models.AnalysisRequest) {
	update := func(fn func(*models.TaskStatus)) {
		h.tasksMu.Lock()
		if s, ok := h.tasks[taskID]; ok {
			fn(s)
		}
		h.tasksMu.Unlock()
	}

	setError := func(msg string) {
		update(func(s *models.TaskStatus) {
			s.Phase = "error"
			s.Error = msg
		})
	}

	update(func(s *models.TaskStatus) { s.Phase = "tracks" })
	log.Printf("[task:%s] fetching tracks for %s (%s to %s)",
		taskID, req.Username, req.From, req.To)

	lastfm := services.NewLastFM(h.cfg.LastFMKey)
	tracks, totalScrobbles, err := lastfm.GetTracks(req.Username, req.From, req.To, req.MaxTracks)
	if err != nil {
		setError(err.Error())
		return
	}

	if len(tracks) == 0 {
		setError("No tracks found for this period")
		return
	}

	update(func(s *models.TaskStatus) {
		s.TotalTracks = len(tracks)
	})

	update(func(s *models.TaskStatus) { s.Phase = "lyrics" })
	log.Printf("[task:%s] searching lyrics for %d tracks", taskID, len(tracks))

	lyricsSvc := services.NewLyrics(h.cfg.GeniusToken, h.cache)

	lyricsMap := lyricsSvc.FetchAll(tracks, 10, func(processed, found int, current string) {
		update(func(s *models.TaskStatus) {
			s.ProcessedTracks = processed
			s.LyricsFound = found
			s.Progress = processed * 100 / len(tracks)
			s.CurrentTrack = current
		})
	})

	log.Printf("[task:%s] lyrics found: %d/%d", taskID, len(lyricsMap), len(tracks))

	if len(lyricsMap) == 0 {
		setError("Could not find lyrics for any track")
		return
	}

	update(func(s *models.TaskStatus) { s.Phase = "analyzing" })

	words, uniqueWords, totalWords := services.AnalyzeWords(lyricsMap, req.ExcludeStopWords)

	update(func(s *models.TaskStatus) {
		s.Phase = "done"
		s.Progress = 100
		s.Result = &models.TaskResult{
			TotalScrobbles:   totalScrobbles,
			UniqueTracks:     len(tracks),
			LyricsFound:      len(lyricsMap),
			LyricsMissing:    len(tracks) - len(lyricsMap),
			TotalUniqueWords: uniqueWords,
			TotalWordCount:   totalWords,
			Words:            words,
		}
	})

	log.Printf("[task:%s] done! top word: %s (%d)",
		taskID, words[0].Word, words[0].Count)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func extractLastSegment(path string) string {
	path = strings.TrimRight(path, "/")
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return path
	}
	return path[i+1:]
}
