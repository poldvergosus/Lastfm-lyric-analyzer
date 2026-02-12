package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"lastfm-lyrics/cache"
	"lastfm-lyrics/config"
	"lastfm-lyrics/handlers"
	"lastfm-lyrics/services"
)

func main() {
	cfg := config.Load()

	if cfg.LastFMKey == "" {
		log.Fatal("LASTFM_API_KEY is required. Set it in .env file.")
	}

	lyricsCache, err := cache.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Cache init failed: %v", err)
	}
	defer lyricsCache.Close()

	total, found := lyricsCache.Stats()
	log.Printf("Cache: %d entries, %d with lyrics", total, found)

	services.LoadStopWords(
		"./data/stopwords-en.json",
		"./data/stopwords-ru.json",
		"./data/stopwords-custom.json",
	)

	h := handlers.New(cfg, lyricsCache)

	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", cfg.AllowOrigins)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(200)
				return
			}
			next(w, r)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/analyze", cors(h.Analyze))
	mux.HandleFunc("/api/status/", cors(h.Status))
	mux.HandleFunc("/api/health", cors(h.Health))

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		log.Println("⏹️  Shutting down...")
		lyricsCache.Close()
		os.Exit(0)
	}()

	addr := ":" + cfg.Port
	log.Printf("Server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
