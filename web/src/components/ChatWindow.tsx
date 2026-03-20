import { useState } from "react";
import { useStream } from "../hooks/useStream";

interface Message {
  role: "user" | "assistant";
  content: string;
}

interface Props {
  model: string;
  sessionId: string;
}

export function ChatWindow({ model, sessionId }: Props) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const { send, buffer, streaming } = useStream(model, sessionId);

  function submit() {
    if (!input.trim()) return;
    const updated = [...messages, { role: "user" as const, content: input }];
    setMessages(updated);
    setInput("");
    send(updated);
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.map((m, i) => (
          <div key={i} className={m.role === "user" ? "text-right" : "text-left"}>
            <span className={`inline-block px-3 py-2 rounded-lg text-sm ${
              m.role === "user" ? "bg-blue-500 text-white" : "bg-gray-100"
            }`}>
              {m.content}
            </span>
          </div>
        ))}
        {streaming && (
          <div className="text-left">
            <span className="inline-block px-3 py-2 rounded-lg text-sm bg-gray-100">
              {buffer}▍
            </span>
          </div>
        )}
      </div>

      <div className="border-t p-3 flex gap-2">
        <input
          className="flex-1 border rounded px-3 py-2 text-sm"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && submit()}
          placeholder="输入消息…"
          disabled={streaming}
        />
        <button
          onClick={submit}
          disabled={streaming}
          className="bg-blue-500 text-white px-4 py-2 rounded text-sm disabled:opacity-50"
        >
          发送
        </button>
      </div>
    </div>
  );
}
