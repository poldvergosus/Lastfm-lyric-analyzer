package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"lastfm-lyrics/models"
)

type LastFM struct {
	apiKey string
	client *http.Client
}

func NewLastFM(apiKey string) *LastFM {
	return &LastFM{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type lfmResponse struct {
	RecentTracks struct {
		Tracks json.RawMessage `json:"track"`
		Attr   struct {
			TotalPages string `json:"totalPages"`
		} `json:"@attr"`
	} `json:"recenttracks"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

type lfmTrack struct {
	Artist struct {
		Name string `json:"#text"`
	} `json:"artist"`
	Name string `json:"name"`
	Attr *struct {
		NowPlaying string `json:"nowplaying"`
	} `json:"@attr"`
}

func (s *LastFM) GetTracks(username, from, to string, maxTracks int) ([]models.Track, int, error) {

	fromTs, err := toTimestamp(from)
	if err != nil {
		return nil, 0, fmt.Errorf("bad 'from' date: %w", err)
	}
	toTs, err := toTimestamp(to)
	if err != nil {
		return nil, 0, fmt.Errorf("bad 'to' date: %w", err)
	}

	type raw struct {
		artist string
		title  string
	}
	var all []raw

	seen := make(map[string]bool)

	page := 1
	totalPages := 1

	for page <= totalPages && page <= 100 {
		params := url.Values{
			"method":  {"user.getrecenttracks"},
			"user":    {username},
			"api_key": {s.apiKey},
			"format":  {"json"},
			"from":    {strconv.FormatInt(fromTs, 10)},
			"to":      {strconv.FormatInt(toTs, 10)},
			"limit":   {"200"},
			"page":    {strconv.Itoa(page)},
		}

		apiURL := "https://ws.audioscrobbler.com/2.0/?" + params.Encode()

		resp, err := s.client.Get(apiURL)
		if err != nil {
			return nil, 0, fmt.Errorf("lastfm request failed: %w", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var data lfmResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, 0, fmt.Errorf("lastfm parse error: %w", err)
		}

		if data.Error != 0 {
			return nil, 0, fmt.Errorf("lastfm: %s", data.Message)
		}

		tp, _ := strconv.Atoi(data.RecentTracks.Attr.TotalPages)
		totalPages = tp

		var pageTracks []lfmTrack
		if err := json.Unmarshal(data.RecentTracks.Tracks, &pageTracks); err != nil {
			var single lfmTrack
			if err2 := json.Unmarshal(data.RecentTracks.Tracks, &single); err2 != nil {
				log.Printf("[lastfm] warning: could not parse tracks on page %d", page)
				page++
				continue
			}
			pageTracks = []lfmTrack{single}
		}

		for _, track := range pageTracks {
			if track.Attr != nil && track.Attr.NowPlaying == "true" {
				continue
			}
			if track.Artist.Name != "" && track.Name != "" {
				all = append(all, raw{track.Artist.Name, track.Name})

				key := strings.ToLower(track.Artist.Name + "|||" + track.Name)
				seen[key] = true
			}
		}

		log.Printf("[lastfm] page %d/%d — %d tracks, %d unique",
			page, totalPages, len(all), len(seen))

		if len(seen) >= maxTracks {
			log.Printf("[lastfm] reached %d unique tracks, stopping early", maxTracks)
			break
		}

		page++
		time.Sleep(200 * time.Millisecond)
	}

	totalScrobbles := len(all)

	type counted struct {
		artist string
		title  string
		count  int
	}
	counts := make(map[string]*counted)

	for _, t := range all {
		key := strings.ToLower(t.artist + "|||" + t.title)
		if c, ok := counts[key]; ok {
			c.count++
		} else {
			counts[key] = &counted{t.artist, t.title, 1}
		}
	}

	tracks := make([]models.Track, 0, len(counts))
	for _, c := range counts {
		tracks = append(tracks, models.Track{
			Artist:    c.artist,
			Title:     c.title,
			PlayCount: c.count,
		})
	}

	sort.Slice(tracks, func(i, j int) bool {
		if tracks[i].PlayCount != tracks[j].PlayCount {
			return tracks[i].PlayCount > tracks[j].PlayCount
		}
		if tracks[i].Artist != tracks[j].Artist {
			return tracks[i].Artist < tracks[j].Artist
		}
		return tracks[i].Title < tracks[j].Title
	})

	if len(tracks) > maxTracks {
		tracks = tracks[:maxTracks]
	}

	log.Printf("[lastfm] %s: %d scrobbles, %d unique tracks (limited to %d)",
		username, totalScrobbles, len(tracks), maxTracks)

	return tracks, totalScrobbles, nil
}

func toTimestamp(date string) (int64, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}
