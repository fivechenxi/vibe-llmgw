import { useState } from "react";

export function LoginPage() {
  const [username, setUsername] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function login() {
    if (!username.trim()) return;
    setError("");
    setLoading(true);
    try {
      const res = await fetch("/auth/dev-login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: username.trim() }),
      });
      const text = await res.text();
      if (!res.ok) {
        let msg = `${res.status} 错误`;
        try {
          const data = JSON.parse(text);
          if (data.error) msg = data.error;
        } catch {
          if (text) msg = `${res.status}: ${text.slice(0, 100)}`;
        }
        setError(msg);
        return;
      }
      const data = JSON.parse(text);
      localStorage.setItem("token", data.token);
      window.location.href = "/";
    } catch (e) {
      setError(`请求失败: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex items-center justify-center h-screen bg-gray-50">
      <div className="bg-white rounded-xl shadow p-8 flex flex-col items-center gap-4 w-80">
        <h1 className="text-xl font-semibold">LLM Gateway</h1>
        <p className="text-sm text-gray-500 text-center">企业内部 AI 聊天平台</p>
        <input
          className="w-full border rounded px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-300"
          placeholder="用户名（如 alice）"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && login()}
          autoFocus
        />
        {error && <p className="text-red-500 text-xs self-start">{error}</p>}
        <button
          onClick={login}
          disabled={loading || !username.trim()}
          className="w-full bg-blue-500 text-white py-2 rounded-lg text-sm hover:bg-blue-600 disabled:opacity-50"
        >
          {loading ? "登录中…" : "登录"}
        </button>
      </div>
    </div>
  );
}
