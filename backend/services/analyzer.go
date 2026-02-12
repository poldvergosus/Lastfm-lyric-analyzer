package services

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"lastfm-lyrics/models"
)

var (
	sectionRe = regexp.MustCompile(`\[.*?\]`)
	wordRe    = regexp.MustCompile(`[a-zA-Zа-яА-ЯёЁ']+`)
)

var stopWords map[string]bool

func LoadStopWords(paths ...string) {
	stopWords = make(map[string]bool)

	for _, path := range paths {
		count := loadOneFile(path)
		log.Printf("[analyzer] loaded %d words from %s", count, path)
	}

	log.Printf("[analyzer] total stop words: %d", len(stopWords))
}

func loadOneFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[analyzer] could not read %s: %v", path, err)
		return 0
	}

	var words []string
	if err := json.Unmarshal(data, &words); err != nil {
		log.Printf("[analyzer] could not parse %s: %v", path, err)
		return 0
	}

	count := 0
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			stopWords[w] = true
			count++
		}
	}

	return count
}

func AnalyzeWords(lyricsMap map[string]string, excludeStop bool) ([]models.WordCount, int, int) {
	counts := make(map[string]int)
	wordTracks := make(map[string]map[string]bool)

	for trackName, text := range lyricsMap {
		text = sectionRe.ReplaceAllString(text, "")
		words := wordRe.FindAllString(strings.ToLower(text), -1)

		for _, w := range words {
			if utf8.RuneCountInString(w) <= 1 {
				continue
			}
			if excludeStop && stopWords[w] {
				continue
			}
			counts[w]++

			if wordTracks[w] == nil {
				wordTracks[w] = make(map[string]bool)
			}
			wordTracks[w][trackName] = true
		}
	}

	result := make([]models.WordCount, 0, len(counts))
	totalWords := 0

	for word, count := range counts {
		tracks := make([]string, 0, len(wordTracks[word]))
		for track := range wordTracks[word] {
			tracks = append(tracks, track)
		}
		sort.Strings(tracks)

		result = append(result, models.WordCount{
			Word:   word,
			Count:  count,
			Tracks: tracks,
		})
		totalWords += count
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Word < result[j].Word
	})

	if len(result) > 300 {
		result = result[:300]
	}

	return result, len(counts), totalWords
}
