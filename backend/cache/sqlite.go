package cache

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type LyricsCache struct {
	db *sql.DB
	mu sync.RWMutex
}

type Entry struct {
	Lyrics string
	Source string
	Found  bool
}

func New(dbPath string) (*LyricsCache, error) {

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS lyrics (
			artist     TEXT NOT NULL,
			title      TEXT NOT NULL,
			lyrics     TEXT,
			source     TEXT,
			found      INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (artist, title)
		)
	`)
	if err != nil {
		return nil, err
	}

	log.Println("[cache] SQLite initialized at", dbPath)
	return &LyricsCache{db: db}, nil
}

func (c *LyricsCache) Get(artist, title string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var lyrics sql.NullString
	var source sql.NullString
	var found int

	err := c.db.QueryRow(
		"SELECT lyrics, source, found FROM lyrics WHERE artist = ? AND title = ?",
		normalize(artist), normalize(title),
	).Scan(&lyrics, &source, &found)

	if err != nil {
		return nil, false
	}

	return &Entry{
		Lyrics: lyrics.String,
		Source: source.String,
		Found:  found == 1,
	}, true
}

func (c *LyricsCache) Set(artist, title, lyrics, source string, found bool) {

	c.mu.Lock()
	defer c.mu.Unlock()

	foundInt := 0
	if found {
		foundInt = 1
	}

	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO lyrics (artist, title, lyrics, source, found)
		 VALUES (?, ?, ?, ?, ?)`,
		normalize(artist), normalize(title), lyrics, source, foundInt,
	)
	if err != nil {
		log.Printf("[cache] write error: %v", err)
	}
}

func (c *LyricsCache) Stats() (total int, found int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.db.QueryRow("SELECT COUNT(*) FROM lyrics").Scan(&total)
	c.db.QueryRow("SELECT COUNT(*) FROM lyrics WHERE found = 1").Scan(&found)
	return
}

func (c *LyricsCache) Close() error {
	return c.db.Close()
}

func normalize(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]

		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
