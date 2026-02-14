import { useState } from "react";
import { useAnalysis } from "./hooks/useAnalysis";
import type { WordCount } from "./types";
import "./App.css";

const translations = {
  en: {
    title: "Last.fm Lyrics Analyzer",
    username: "Last.fm username",
    from: "From",
    to: "To",
    maxTracks: "Max tracks",
    analyze: "Analyze",
    analyzing: "Analyzing...",
    fetchingTracks: "Fetching tracks from Last.fm...",
    fetchingLyrics: "Fetching lyrics",
    found: "found",
    analyzingWords: "Analyzing words...",
    scrobbles: "Scrobbles",
    tracks: "Tracks",
    lyricsFound: "Lyrics found",
    coverage: "Coverage",
    uniqueWords: "Unique words",
    word: "Word",
    count: "Count",
    tracksCol: "Tracks",
    timesIn: "times in",
    tracksLabel: "tracks:",
  },
  ru: {
    title: "Анализ текстов Last.fm",
    username: "Имя пользователя Last.fm",
    from: "От",
    to: "До",
    maxTracks: "Макс. треков",
    analyze: "Анализировать",
    analyzing: "Анализируем...",
    fetchingTracks: "Загружаем треки с Last.fm...",
    fetchingLyrics: "Ищем тексты",
    found: "найдено",
    analyzingWords: "Анализируем слова...",
    scrobbles: "Прослушиваний",
    tracks: "Треков",
    lyricsFound: "Текстов найдено",
    coverage: "Покрытие",
    uniqueWords: "Уникальных слов",
    word: "Слово",
    count: "Кол-во",
    tracksCol: "Треки",
    timesIn: "раз в",
    tracksLabel: "треках:",
  },
};

type Lang = "en" | "ru";

function App() {
  const { state, result, run } = useAnalysis();

  const [lang, setLang] = useState<Lang>("en");
  const [username, setUsername] = useState("");
  const [from, setFrom] = useState("2024-01-01");
  const [to, setTo] = useState(new Date().toISOString().split("T")[0]);
  const [maxTracks, setMaxTracks] = useState(200);
  const [selectedWord, setSelectedWord] = useState<WordCount | null>(null);
  const [expandedTrack, setExpandedTrack] = useState<string | null>(null);
  const [view, setView] = useState<"cloud" | "table">("cloud");

  const t = translations[lang];
  const isRunning = state.phase === "running";

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (username.trim()) {
      setSelectedWord(null);
      run(username.trim(), from, to, maxTracks);
    }
  };

  const handleWordClick = (w: WordCount) => {
    setSelectedWord(selectedWord?.word === w.word ? null : w);
  };

    const highlightWord = (line: string, word: string) => {
    const regex = new RegExp(`(${word})`, "gi");
    const parts = line.split(regex);
    return parts.map((part, i) =>
      part.toLowerCase() === word.toLowerCase() ? (
        <mark key={i}>{part}</mark>
      ) : (
        <span key={i}>{part}</span>
      )
    );
  };

  return (
    <div className={result || isRunning || state.phase === "error" ? "app" : "app-centered"}>
      
      <button
        className="lang-toggle"
        onClick={() => setLang(lang === "en" ? "ru" : "en")}
      >
        {lang === "en" ? "RU" : "EN"}
      </button>

      <h1>{t.title}</h1>

      <form onSubmit={handleSubmit} className="form">
        <input
          type="text"
          placeholder={t.username}
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          disabled={isRunning}
          required
        />
        <div className="form-row">
          <label>
            {t.from}
            <input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              disabled={isRunning}
            />
          </label>
          <label>
            {t.to}
            <input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              disabled={isRunning}
            />
          </label>
          <label>
            {t.maxTracks}
            <input
              type="number"
              value={maxTracks}
              onChange={(e) => setMaxTracks(Number(e.target.value))}
              disabled={isRunning}
              min={10}
              max={2000}
            />
          </label>
        </div>
        <button type="submit" disabled={isRunning || !username.trim()}>
          {isRunning ? t.analyzing : t.analyze}
        </button>
      </form>

      {isRunning && (
        <div className="progress">
          <div className="progress-bar">
            <div
              className="progress-fill"
              style={{ width: `${state.progress}%` }}
            />
          </div>
          <p>
            {state.backendPhase === "tracks" && t.fetchingTracks}
            {state.backendPhase === "lyrics" &&
              `${t.fetchingLyrics}: ${state.processedTracks}/${state.totalTracks} (${t.found}: ${state.lyricsFound})`}
            {state.backendPhase === "analyzing" && t.analyzingWords}
          </p>
          {state.currentTrack && (
            <p className="current-track">{state.currentTrack}</p>
          )}
        </div>
      )}
      {state.phase === "error" && (
        <div className="error">{state.error}</div>
      )}

      {result && (
        <div className="results">
          <div className="stats">
            <div className="stat">
              <span className="stat-value">{result.total_scrobbles}</span>
              <span className="stat-label">{t.scrobbles}</span>
            </div>
            <div className="stat">
              <span className="stat-value">{result.unique_tracks}</span>
              <span className="stat-label">{t.tracks}</span>
            </div>
            <div className="stat">
              <span className="stat-value">{result.lyrics_found}</span>
              <span className="stat-label">{t.lyricsFound}</span>
            </div>
            <div className="stat">
              <span className="stat-value">
                {Math.round(
                  (result.lyrics_found / result.unique_tracks) * 100
                )}%
              </span>
              <span className="stat-label">{t.coverage}</span>
            </div>
            <div className="stat">
              <span className="stat-value">{result.total_unique_words}</span>
              <span className="stat-label">{t.uniqueWords}</span>
            </div>
          </div>

  <div className="view-toggle">
            <button
              className={view === "cloud" ? "active" : ""}
              onClick={() => setView("cloud")}
            >
              {lang === "ru" ? "Облако" : "Cloud"}
            </button>
            <button
              className={view === "table" ? "active" : ""}
              onClick={() => setView("table")}
            >
              {lang === "ru" ? "Таблица" : "Table"}
            </button>
          </div>

          {view === "cloud" && (
            <div className="word-cloud">
              {result.words.slice(0, 80).map((w: WordCount) => {
                const max = result.words[0].count;
                const size = 14 + (w.count / max) * 36;
                const opacity = 0.4 + (w.count / max) * 0.6;
                const isSelected = selectedWord?.word === w.word;
                return (
                  <span
                    key={w.word}
                    className={`word ${isSelected ? "word-selected" : ""}`}
                    style={{ fontSize: `${size}px`, opacity: isSelected ? 1 : opacity }}
                    title={`${w.word}: ${w.count}`}
                    onClick={() => handleWordClick(w)}
                  >
                    {w.word}
                  </span>
                );
              })}
            </div>
          )}

          {selectedWord && (
            <div className="track-list">
              <h3>
                "{selectedWord.word}" — {selectedWord.count} {t.timesIn}{" "}
                {selectedWord.tracks.length} {t.tracksLabel}
              </h3>
              <ul>
                {selectedWord.tracks.map((track) => (
                  <li key={track}>
                    <div
                      className="track-name"
                      onClick={() =>
                        setExpandedTrack(
                          expandedTrack === track ? null : track
                        )
                      }
                    >
                      {track}
                      {result.lyrics?.[track] && (
                        <span className="expand-icon">
                          {expandedTrack === track ? "▲" : "▼"}
                        </span>
                      )}
                    </div>
                    {expandedTrack === track && result.lyrics?.[track] && (
                      <div className="track-lyrics">
                        {result.lyrics[track].split("\n").map((line, i) => (
                          <p key={i}>
                            {highlightWord(line, selectedWord.word)}
                          </p>
                        ))}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {view === "table" && (
            <table className="word-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>{t.word}</th>
                  <th>{t.count}</th>
                  <th>{t.tracksCol}</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {result.words.slice(0, 50).map((w: WordCount, i: number) => (
                  <tr
                    key={w.word}
                    className={`table-row ${selectedWord?.word === w.word ? "row-selected" : ""}`}
                    onClick={() => handleWordClick(w)}
                  >
                    <td>{i + 1}</td>
                    <td>{w.word}</td>
                    <td>{w.count}</td>
                    <td>{w.tracks.length}</td>
                    <td>
                      <div
                        className="bar"
                        style={{
                          width: `${(w.count / result.words[0].count) * 100}%`,
                        }}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}

export default App;