package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"lastfm-lyrics/models"
)

type MusicBrainz struct {
	client *http.Client
}

func NewMusicBrainz() *MusicBrainz {
	return &MusicBrainz{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (mb *MusicBrainz) mbRequest(url string) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "LastFmLyricsAnalyzer/1.0 (contact@example.com)")
	req.Header.Set("Accept", "application/json")

	resp, err := mb.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 503 {

		time.Sleep(2 * time.Second)
		return mb.mbRequest(url)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("musicbrainz: HTTP %d", resp.StatusCode)
	}

	return body, nil
}

func (mb *MusicBrainz) FindArtist(name string) (string, string, error) {
	params := url.Values{
		"query": {name},
		"limit": {"5"},
		"fmt":   {"json"},
	}

	body, err := mb.mbRequest("https://musicbrainz.org/ws/2/artist/?" + params.Encode())
	if err != nil {
		return "", "", err
	}

	var result struct {
		Artists []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artists"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}

	if len(result.Artists) == 0 {
		return "", "", fmt.Errorf("artist not found: %s", name)
	}

	artist := result.Artists[0]
	log.Printf("[musicbrainz] found artist: %s (ID: %s)", artist.Name, artist.ID)
	return artist.ID, artist.Name, nil
}

func (mb *MusicBrainz) GetDiscography(artistID, artistName string, maxTracks int) ([]models.Track, error) {
	releaseGroups, err := mb.getReleaseGroups(artistID)
	if err != nil {
		return nil, err
	}

	log.Printf("[musicbrainz] %s: %d release groups", artistName, len(releaseGroups))

	seen := make(map[string]bool)
	var tracks []models.Track

	for _, rg := range releaseGroups {
		if len(tracks) >= maxTracks {
			break
		}

		time.Sleep(1100 * time.Millisecond)

		rgTracks, err := mb.getTracksFromReleaseGroup(rg.ID)
		if err != nil {
			log.Printf("[musicbrainz] warning: %v", err)
			continue
		}

		for _, title := range rgTracks {
			key := normalizeTitle(title)
			if seen[key] {
				continue
			}
			seen[key] = true

			tracks = append(tracks, models.Track{
				Artist:    artistName,
				Title:     title,
				PlayCount: 0,
			})

			if len(tracks) >= maxTracks {
				break
			}
		}

		log.Printf("[musicbrainz] %s: %d unique tracks so far (from %s)",
			artistName, len(tracks), rg.Title)
	}

	log.Printf("[musicbrainz] %s: %d total unique tracks", artistName, len(tracks))
	return tracks, nil
}

type releaseGroup struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"primary-type"`
}

func (mb *MusicBrainz) getReleaseGroups(artistID string) ([]releaseGroup, error) {
	var all []releaseGroup
	offset := 0
	limit := 100

	for {
		params := url.Values{
			"artist": {artistID},
			"limit":  {fmt.Sprintf("%d", limit)},
			"offset": {fmt.Sprintf("%d", offset)},
			"fmt":    {"json"},
		}

		body, err := mb.mbRequest("https://musicbrainz.org/ws/2/release-group?" + params.Encode())
		if err != nil {
			return nil, err
		}

		var result struct {
			ReleaseGroups []releaseGroup `json:"release-groups"`
			Count         int            `json:"release-group-count"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		all = append(all, result.ReleaseGroups...)

		if len(all) >= result.Count || len(result.ReleaseGroups) == 0 {
			break
		}

		offset += limit
		time.Sleep(1100 * time.Millisecond)
	}

	return all, nil
}

func (mb *MusicBrainz) getTracksFromReleaseGroup(rgID string) ([]string, error) {

	params := url.Values{
		"release-group": {rgID},
		"limit":         {"1"},
		"inc":           {"recordings"},
		"fmt":           {"json"},
	}

	body, err := mb.mbRequest("https://musicbrainz.org/ws/2/release?" + params.Encode())
	if err != nil {
		return nil, err
	}

	var result struct {
		Releases []struct {
			Media []struct {
				Tracks []struct {
					Title string `json:"title"`
				} `json:"tracks"`
			} `json:"media"`
		} `json:"releases"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var titles []string
	if len(result.Releases) > 0 {
		for _, media := range result.Releases[0].Media {
			for _, track := range media.Tracks {
				if track.Title != "" {
					titles = append(titles, track.Title)
				}
			}
		}
	}

	return titles, nil
}

func normalizeTitle(s string) string {
	s = cleanTitle(s)
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result = append(result, c)
	}
	return string(result)
}
