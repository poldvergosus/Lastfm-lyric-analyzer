const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export async function startAnalysis(
  username: string,
  from: string,
  to: string,
  maxTracks: number = 500
): Promise<string> {
  const resp = await fetch(`${API_URL}/api/analyze`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      username,
      from,
      to,
      max_tracks: maxTracks,
    }),
  });

  if (!resp.ok) {
    const err = await resp.json();
    throw new Error(err.error || "Request failed");
  }

  const data = await resp.json();
  return data.task_id;
}

export async function getStatus(taskId: string) {
  const resp = await fetch(`${API_URL}/api/status/${taskId}`);

  if (!resp.ok) {
    throw new Error("Failed to get status");
  }

  return resp.json();
}