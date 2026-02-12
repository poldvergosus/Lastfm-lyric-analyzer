package services

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"lastfm-lyrics/cache"
	"lastfm-lyrics/models"

	"golang.org/x/net/html"
)

type Lyrics struct {
	geniusToken string
	cache       *cache.LyricsCache
	client      *http.Client
}

func NewLyrics(geniusToken string, c *cache.LyricsCache) *Lyrics {
	return &Lyrics{
		geniusToken: geniusToken,
		cache:       c,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Lyrics) FetchAll(
	tracks []models.Track,
	workers int,
	progressFn func(processed, found int, current string),
) map[string]string {

	type job struct {
		track models.Track
	}
	type result struct {
		key    string
		lyrics string
		found  bool
	}

	jobs := make(chan job, len(tracks))
	results := make(chan result, len(tracks))

	for w := 0; w < workers; w++ {
		go func() {
			for j := range jobs {
				lyrics, found, _ := s.fetchOne(j.track.Artist, j.track.Title)
				key := j.track.Artist + " — " + j.track.Title
				results <- result{key: key, lyrics: lyrics, found: found}
			}
		}()
	}

	for _, t := range tracks {
		jobs <- job{track: t}
	}
	close(jobs)

	lyricsMap := make(map[string]string)
	processed := 0
	found := 0

	for range tracks {
		r := <-results
		processed++
		if r.found {
			lyricsMap[r.key] = r.lyrics
			found++
		}
		if progressFn != nil {
			progressFn(processed, found, r.key)
		}
	}

	return lyricsMap
}

func (s *Lyrics) fetchOne(artist, title string) (string, bool, string) {
	cleaned := cleanTitle(title)

	if entry, ok := s.cache.Get(artist, cleaned); ok {
		return entry.Lyrics, entry.Found, "cache"
	}

	if lyrics, ok := s.tryLrclib(artist, cleaned); ok {
		log.Printf("[lyrics] ✅ lrclib: %s — %s", artist, cleaned)
		s.cache.Set(artist, cleaned, lyrics, "lrclib", true)
		return lyrics, true, "lrclib"
	}

	if s.geniusToken != "" {
		if lyrics, ok := s.tryGenius(artist, cleaned); ok {
			log.Printf("[lyrics] ✅ genius: %s — %s", artist, cleaned)
			s.cache.Set(artist, cleaned, lyrics, "genius", true)
			return lyrics, true, "genius"
		}
	}

	log.Printf("[lyrics] ❌ not found: %s — %s", artist, cleaned)
	s.cache.Set(artist, cleaned, "", "none", false)
	return "", false, ""
}

type lrclibResult struct {
	PlainLyrics string `json:"plainLyrics"`
}

func (s *Lyrics) tryLrclib(artist, title string) (string, bool) {
	params := url.Values{
		"artist_name": {artist},
		"track_name":  {title},
	}

	req, _ := http.NewRequest("GET",
		"https://lrclib.net/api/search?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "LastFmLyricsAnalyzer/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var results []lrclibResult
	if err := json.Unmarshal(body, &results); err != nil {
		return "", false
	}

	for _, r := range results {
		if r.PlainLyrics != "" {
			return r.PlainLyrics, true
		}
	}
	return "", false
}

type geniusSearch struct {
	Response struct {
		Hits []struct {
			Result struct {
				URL string `json:"url"`
			} `json:"result"`
		} `json:"hits"`
	} `json:"response"`
}

func (s *Lyrics) tryGenius(artist, title string) (string, bool) {

	query := artist + " " + title
	params := url.Values{"q": {query}}

	req, _ := http.NewRequest("GET",
		"https://api.genius.com/search?"+params.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+s.geniusToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var search geniusSearch
	if err := json.Unmarshal(body, &search); err != nil {
		return "", false
	}

	if len(search.Response.Hits) == 0 {
		return "", false
	}

	songURL := search.Response.Hits[0].Result.URL

	time.Sleep(300 * time.Millisecond)

	pageReq, _ := http.NewRequest("GET", songURL, nil)
	pageReq.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	pageResp, err := s.client.Do(pageReq)
	if err != nil {
		return "", false
	}
	defer pageResp.Body.Close()

	lyrics := parseGeniusHTML(pageResp.Body)
	if lyrics == "" {
		return "", false
	}
	return lyrics, true
}

func parseGeniusHTML(r io.Reader) string {
	doc, err := html.Parse(r)
	if err != nil {
		return ""
	}

	var sb strings.Builder

	var find func(*html.Node)
	find = func(n *html.Node) {

		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "data-lyrics-container" && a.Val == "true" {
					getText(n, &sb)
					sb.WriteString("\n")
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}

	find(doc)
	return strings.TrimSpace(sb.String())
}

func getText(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
	}
	if n.Type == html.ElementNode && n.Data == "br" {
		sb.WriteString("\n")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		getText(c, sb)
	}
}

var (
	reParens = regexp.MustCompile(`\s*[\(\[].*?[\)\]]\s*`)

	reSuffix = regexp.MustCompile(
		`(?i)\s*-\s*(remaster|live|demo|remix|deluxe|bonus|edit|version|` +
			`mix|single|acoustic|instrumental|radio|extended|original).*`)
)

func cleanTitle(title string) string {
	title = reParens.ReplaceAllString(title, " ")
	title = reSuffix.ReplaceAllString(title, "")
	return strings.TrimSpace(title)
}
