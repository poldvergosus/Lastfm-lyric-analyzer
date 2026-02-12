import { useState } from "react";
import { useAnalysis } from "./hooks/useAnalysis";
import type { WordCount } from "./types";
import "./App.css";

function App() {
  const { state, result, run } = useAnalysis();

  const [username, setUsername] = useState("");
  const [from, setFrom] = useState("2024-01-01");
  const [to, setTo] = useState(new Date().toISOString().split("T")[0]);
  const [maxTracks, setMaxTracks] = useState(200);

  const isRunning = state.phase === "running";

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (username.trim()) {
      run(username.trim(), from, to, maxTracks);
    }
  };

  return (
    <div className="app">
      <h1>Last.fm Lyrics Analyzer</h1>

      <form onSubmit={handleSubmit} className="form">
        <input
          type="text"
          placeholder="Last.fm username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          disabled={isRunning}
          required
        />
        <div className="form-row">
          <label>
            From
            <input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              disabled={isRunning}
            />
          </label>
          <label>
            To
            <input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              disabled={isRunning}
            />
          </label>
          <label>
            Max tracks
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
          {isRunning ? "Analyzing..." : "Analyze"}
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
            {state.backendPhase === "tracks" && "Fetching tracks from Last.fm..."}
            {state.backendPhase === "lyrics" &&
              `Fetching lyrics: ${state.processedTracks}/${state.totalTracks} (found: ${state.lyricsFound})`}
            {state.backendPhase === "analyzing" && "Analyzing words..."}
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
              <span className="stat-label">Scrobbles</span>
            </div>
            <div className="stat">
              <span className="stat-value">{result.unique_tracks}</span>
              <span className="stat-label">Tracks</span>
            </div>
            <div className="stat">
              <span className="stat-value">{result.lyrics_found}</span>
              <span className="stat-label">Lyrics found</span>
            </div>
            <div className="stat">
              <span className="stat-value">
                {Math.round((result.lyrics_found / result.unique_tracks) * 100)}%
              </span>
              <span className="stat-label">Coverage</span>
            </div>
            <div className="stat">
              <span className="stat-value">{result.total_unique_words}</span>
              <span className="stat-label">Unique words</span>
            </div>
          </div>

          <div className="word-cloud">
            {result.words.slice(0, 80).map((w: WordCount) => {
              const max = result.words[0].count;
              const size = 14 + (w.count / max) * 36;
              const opacity = 0.4 + (w.count / max) * 0.6;
              return (
                <span
                  key={w.word}
                  className="word"
                  style={{ fontSize: `${size}px`, opacity }}
                  title={`${w.word}: ${w.count}`}
                >
                  {w.word}
                </span>
              );
            })}
          </div>

          <table className="word-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Word</th>
                <th>Count</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {result.words.slice(0, 50).map((w: WordCount, i: number) => (
                <tr key={w.word}>
                  <td>{i + 1}</td>
                  <td>{w.word}</td>
                  <td>{w.count}</td>
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
        </div>
      )}
    </div>
  );
}

export default App;