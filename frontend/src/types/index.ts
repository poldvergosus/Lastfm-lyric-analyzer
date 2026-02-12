export interface WordCount {
  word: string;
  count: number;
  tracks: string[];
}

export interface TaskResult {
  total_scrobbles: number;
  unique_tracks: number;
  lyrics_found: number;
  lyrics_missing: number;
  total_unique_words: number;
  total_word_count: number;
  words: WordCount[];
}

export interface TaskStatus {
  id: string;
  phase: string;
  progress: number;
  current_track: string;
  total_tracks: number;
  processed_tracks: number;
  lyrics_found: number;
  error?: string;
  result?: TaskResult;
}