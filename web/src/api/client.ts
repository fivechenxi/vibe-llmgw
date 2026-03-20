const BASE = "/api";

function authHeader(): HeadersInit {
  const token = localStorage.getItem("token");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export async function getModels() {
  const res = await fetch(`${BASE}/models`, { headers: authHeader() });
  return res.json();
}

export async function getQuota() {
  const res = await fetch(`${BASE}/quota`, { headers: authHeader() });
  return res.json();
}

export async function getSessions() {
  const res = await fetch(`${BASE}/sessions`, { headers: authHeader() });
  return res.json();
}

export async function getSession(sessionId: string) {
  const res = await fetch(`${BASE}/sessions/${sessionId}`, { headers: authHeader() });
  return res.json();
}

export function chatStream(
  model: string,
  messages: { role: string; content: string }[],
  sessionId: string,
  onChunk: (text: string) => void,
  onDone: () => void,
) {
  fetch(`${BASE}/chat`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeader() },
    body: JSON.stringify({ model, messages, session_id: sessionId, stream: true }),
  }).then(async (res) => {
    const reader = res.body!.getReader();
    const decoder = new TextDecoder();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      onChunk(decoder.decode(value));
    }
    onDone();
  });
}
