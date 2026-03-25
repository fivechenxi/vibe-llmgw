const BASE = "/api";

function authHeader(): HeadersInit {
  const token = localStorage.getItem("token");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function fetchJSON(url: string, init?: RequestInit) {
  const res = await fetch(url, init);
  if (res.status === 401) {
    logout();
    throw new Error("unauthorized");
  }
  return res.json();
}

export function logout() {
  localStorage.removeItem("token");
  window.location.href = "/login";
}

export async function getModels() {
  return fetchJSON(`${BASE}/models`, { headers: authHeader() });
}

export async function getQuota() {
  return fetchJSON(`${BASE}/quota`, { headers: authHeader() });
}

export async function getSessions() {
  return fetchJSON(`${BASE}/sessions`, { headers: authHeader() });
}

export async function getSession(sessionId: string) {
  return fetchJSON(`${BASE}/sessions/${sessionId}`, { headers: authHeader() });
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
    if (res.status === 401) {
      logout();
      return;
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
      onChunk("[ERROR] " + (err.error ?? `HTTP ${res.status}`));
      onDone();
      return;
    }
    const reader = res.body!.getReader();
    const decoder = new TextDecoder();
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      onChunk(decoder.decode(value, { stream: true }));
    }
    onDone();
  }).catch(() => {
    onChunk("[ERROR] 网络异常，请重试");
    onDone();
  });
}
