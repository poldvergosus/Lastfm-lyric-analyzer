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
		Tracks []lfmTrack `json:"track"`
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

func (s *LastFM) GetTracks(username, from, to string) ([]models.Track, int, error) {

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

		for _, t := range data.RecentTracks.Tracks {

			if t.Attr != nil && t.Attr.NowPlaying == "true" {
				continue
			}

			if t.Artist.Name != "" && t.Name != "" {
				all = append(all, raw{t.Artist.Name, t.Name})
			}
		}

		log.Printf("[lastfm] page %d/%d — %d tracks total", page, totalPages, len(all))
		page++

		time.Sleep(200 * time.Millisecond)
	}

	totalScrobbles := len(all)

	type counted struct {
		artist string
		title  string
		count  int
	}
	seen := make(map[string]*counted)

	for _, t := range all {
		key := strings.ToLower(t.artist + "|||" + t.title)

		if c, ok := seen[key]; ok {
			c.count++
		} else {
			seen[key] = &counted{t.artist, t.title, 1}
		}
	}

	tracks := make([]models.Track, 0, len(seen))
	for _, c := range seen {
		tracks = append(tracks, models.Track{
			Artist:    c.artist,
			Title:     c.title,
			PlayCount: c.count,
		})
	}

	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].PlayCount > tracks[j].PlayCount
	})

	log.Printf("[lastfm] %s: %d scrobbles, %d unique tracks",
		username, totalScrobbles, len(tracks))

	return tracks, totalScrobbles, nil
}

func toTimestamp(date string) (int64, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}
