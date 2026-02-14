package models

type Track struct {
	Artist    string `json:"artist"`
	Title     string `json:"title"`
	PlayCount int    `json:"play_count"`
}

type WordCount struct {
	Word   string   `json:"word"`
	Count  int      `json:"count"`
	Tracks []string `json:"tracks"`
}

type AnalysisRequest struct {
	Username         string `json:"username"`
	From             string `json:"from"`
	To               string `json:"to"`
	MaxTracks        int    `json:"max_tracks"`
	ExcludeStopWords bool   `json:"exclude_stop_words"`
}

type TaskStatus struct {
	ID              string      `json:"id"`
	Phase           string      `json:"phase"`
	Progress        int         `json:"progress"`
	CurrentTrack    string      `json:"current_track"`
	TotalTracks     int         `json:"total_tracks"`
	ProcessedTracks int         `json:"processed_tracks"`
	LyricsFound     int         `json:"lyrics_found"`
	Error           string      `json:"error,omitempty"`
	Result          *TaskResult `json:"result,omitempty"`
}

type TaskResult struct {
	TotalScrobbles   int               `json:"total_scrobbles"`
	UniqueTracks     int               `json:"unique_tracks"`
	LyricsFound      int               `json:"lyrics_found"`
	LyricsMissing    int               `json:"lyrics_missing"`
	TotalUniqueWords int               `json:"total_unique_words"`
	TotalWordCount   int               `json:"total_word_count"`
	Words            []WordCount       `json:"words"`
	Lyrics           map[string]string `json:"lyrics,omitempty"`
}
