import { useEffect, useRef, useState } from "react";
import { getSession } from "../api/client";
import { useStream } from "../hooks/useStream";

interface Message {
  role: "user" | "assistant";
  content: string;
}

interface Props {
  model: string;
  sessionId: string;
  onModelChange?: (model: string) => void;
  onHasMessages?: (hasMessages: boolean) => void;
}

export function ChatWindow({ model, sessionId, onModelChange, onHasMessages }: Props) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const { send, buffer, streaming } = useStream(model, sessionId);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Load history when switching to a different session
  useEffect(() => {
    setMessages([]);
    onHasMessages?.(false);
    getSession(sessionId).then((data) => {
      const msgs = data.messages ?? [];
      setMessages(msgs);
      onHasMessages?.(msgs.length > 0);
      if (data.model) onModelChange?.(data.model);
    }).catch(() => {});
  }, [sessionId]);

  // Auto-scroll to bottom when messages or streaming buffer changes
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, buffer]);

  function submit() {
    if (!input.trim() || !model || streaming) return;
    const userMsg: Message = { role: "user", content: input.trim() };
    const updated = [...messages, userMsg];
    setMessages(updated);
    setInput("");
    onHasMessages?.(true);
    send(updated, (content) => {
      if (content) {
        setMessages((prev) => [...prev, { role: "assistant", content }]);
      }
    });
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {messages.length === 0 && !streaming && (
          <p className="text-center text-gray-400 text-sm mt-16">发送消息开始对话</p>
        )}
        {messages.map((m, i) => (
          <div key={i} className={m.role === "user" ? "text-right" : "text-left"}>
            <span
              className={`inline-block px-3 py-2 rounded-lg text-sm whitespace-pre-wrap max-w-[80%] ${
                m.role === "user" ? "bg-blue-500 text-white" : "bg-gray-100 text-gray-800"
              }`}
            >
              {m.content}
            </span>
          </div>
        ))}
        {streaming && (
          <div className="text-left">
            <span className="inline-block px-3 py-2 rounded-lg text-sm bg-gray-100 text-gray-800 whitespace-pre-wrap max-w-[80%]">
              {buffer || "▍"}
              {buffer && "▍"}
            </span>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <div className="border-t p-3 flex gap-2">
        <input
          className="flex-1 border rounded px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-300"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && submit()}
          placeholder="输入消息，按 Enter 发送"
          disabled={streaming || !model}
        />
        <button
          onClick={submit}
          disabled={streaming || !input.trim() || !model}
          className="bg-blue-500 text-white px-4 py-2 rounded text-sm disabled:opacity-50 hover:bg-blue-600"
        >
          发送
        </button>
      </div>
    </div>
  );
}
