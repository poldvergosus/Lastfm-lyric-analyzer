import { useState, useCallback, useRef } from "react";
import type { TaskStatus, TaskResult } from "../types";
import { startAnalysis, startArtistAnalysis, getStatus } from "../api/client";

export type Phase = "idle" | "running" | "done" | "error";

export interface AnalysisState {
  phase: Phase;
  progress: number;
  currentTrack: string;
  totalTracks: number;
  processedTracks: number;
  lyricsFound: number;
  error: string | null;
  backendPhase: string;
}

export function useAnalysis() {
  const [state, setState] = useState<AnalysisState>({
    phase: "idle",
    progress: 0,
    currentTrack: "",
    totalTracks: 0,
    processedTracks: 0,
    lyricsFound: 0,
    error: null,
    backendPhase: "",
  });

  const [result, setResult] = useState<TaskResult | null>(null);

  const intervalRef = useRef<number | null>(null);

  const stop = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const run = useCallback(
    async (username: string, from: string, to: string, maxTracks: number, lang: string = "en") => {
      setResult(null);
      stop();

      setState({
        phase: "running",
        progress: 0,
        currentTrack: "",
        totalTracks: 0,
        processedTracks: 0,
        lyricsFound: 0,
        error: null,
        backendPhase: "starting",
      });

      try {
      
        const taskId = await startAnalysis(username, from, to, maxTracks, lang);

        intervalRef.current = window.setInterval(async () => {
          try {
            const status: TaskStatus = await getStatus(taskId);

            setState({
              phase: status.phase === "done" ? "done"
                   : status.phase === "error" ? "error"
                   : "running",
              progress: status.progress || 0,
              currentTrack: status.current_track || "",
              totalTracks: status.total_tracks || 0,
              processedTracks: status.processed_tracks || 0,
              lyricsFound: status.lyrics_found || 0,
              error: status.error || null,
              backendPhase: status.phase,
            });

            if (status.phase === "done" && status.result) {
              stop();
              setResult(status.result);
            }

            if (status.phase === "error") {
              stop();
            }
          } catch {
      
          }
        }, 1500);
      } catch (err) {
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: err instanceof Error ? err.message : "Unknown error",
        }));
      }
    },
    [stop]
  );

  const runArtist = useCallback(
    async (artist: string, maxTracks: number, lang: string = "en") => {
      setResult(null);
      stop();

      setState({
        phase: "running",
        progress: 0,
        currentTrack: "",
        totalTracks: 0,
        processedTracks: 0,
        lyricsFound: 0,
        error: null,
        backendPhase: "starting",
      });

      try {
        const taskId = await startArtistAnalysis(artist, maxTracks, lang);

        intervalRef.current = window.setInterval(async () => {
          try {
            const status: TaskStatus = await getStatus(taskId);

            setState({
              phase: status.phase === "done" ? "done"
                   : status.phase === "error" ? "error"
                   : "running",
              progress: status.progress || 0,
              currentTrack: status.current_track || "",
              totalTracks: status.total_tracks || 0,
              processedTracks: status.processed_tracks || 0,
              lyricsFound: status.lyrics_found || 0,
              error: status.error || null,
              backendPhase: status.phase,
            });

            if (status.phase === "done" && status.result) {
              stop();
              setResult(status.result);
            }

            if (status.phase === "error") {
              stop();
            }
          } catch {
          }
        }, 1500);
      } catch (err) {
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: err instanceof Error ? err.message : "Unknown error",
        }));
      }
    },
    [stop]
  );


  return { state, result, run, runArtist, stop };
}