package main

import (
	"fmt"
	"log"

	"lastfm-lyrics/cache"
	"lastfm-lyrics/config"
	"lastfm-lyrics/services"
)

func main() {
	cfg := config.Load()

	if cfg.LastFMKey == "" {
		log.Fatal("LASTFM_API_KEY is not set in .env")
	}

	lyricsCache, err := cache.New(cfg.DBPath)
	if err != nil {
		log.Fatal("Cache error: ", err)
	}
	defer lyricsCache.Close()

	fmt.Println("Fetching tracks from Last.fm...")
	lastfm := services.NewLastFM(cfg.LastFMKey)

	tracks, totalScrobbles, err := lastfm.GetTracks(
		"Poldvergos",
		"2026-02-02",
		"2026-02-12",
	)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	fmt.Printf("Total scrobbles: %d, Unique tracks: %d\n\n", totalScrobbles, len(tracks))

	testTracks := tracks
	if len(testTracks) > 10 {
		testTracks = testTracks[:10]
	}

	fmt.Println("Searching lyrics for top 10 tracks...")
	fmt.Println("──────────────────────────────────────")

	lyricsSvc := services.NewLyrics(cfg.GeniusToken, lyricsCache)

	lyricsMap := lyricsSvc.FetchAll(testTracks, 3, func(processed, found int, current string) {
		fmt.Printf("  [%d/%d] found: %d | %s\n",
			processed, len(testTracks), found, current)
	})

	fmt.Println("──────────────────────────────────────")
	fmt.Printf("\nLyrics found: %d / %d\n\n", len(lyricsMap), len(testTracks))

	for key, lyrics := range lyricsMap {
		fmt.Printf("🎵 %s\n", key)
		preview := lyrics
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("%s\n\n", preview)
		break
	}
}
