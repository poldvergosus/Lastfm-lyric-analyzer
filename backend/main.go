package main

import (
	"fmt"
	"log"

	"lastfm-lyrics/config"
	"lastfm-lyrics/services"
)

func main() {
	cfg := config.Load()

	if cfg.LastFMKey == "" {
		log.Fatal("LASTFM_API_KEY is not set in .env")
	}

	fmt.Println("API key loaded, fetching tracks...")

	lastfm := services.NewLastFM(cfg.LastFMKey)

	tracks, totalScrobbles, err := lastfm.GetTracks(
		"Poldvergos",
		"2026-01-01",
		"2026-02-02",
	)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	fmt.Printf("\nTotal scrobbles: %d\n", totalScrobbles)
	fmt.Printf("Unique tracks: %d\n\n", len(tracks))

	limit := 20
	if len(tracks) < limit {
		limit = len(tracks)
	}

	fmt.Println("Top tracks:")
	for i := 0; i < limit; i++ {
		t := tracks[i]
		fmt.Printf("  %d. %s — %s (%d plays)\n",
			i+1, t.Artist, t.Title, t.PlayCount)
	}
}
