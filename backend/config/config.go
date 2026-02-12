package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port         string
	LastFMKey    string
	GeniusToken  string
	AllowOrigins string
	DBPath       string
}

func Load() *Config {
	loadEnvFile(".env")

	return &Config{
		Port:         getEnv("PORT", "8080"),
		LastFMKey:    getEnv("LASTFM_API_KEY", ""),
		GeniusToken:  getEnv("GENIUS_TOKEN", ""),
		AllowOrigins: getEnv("ALLOW_ORIGINS", "*"),
		DBPath:       getEnv("DB_PATH", "./data/lyrics_cache.db"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		value = strings.Trim(value, `"'`)

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
