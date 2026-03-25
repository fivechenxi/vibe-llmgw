import { useEffect, useState } from "react";
import { v4 as uuidv4 } from "uuid";
import { ChatWindow } from "../components/ChatWindow";
import { ModelSelector } from "../components/ModelSelector";
import { SessionList } from "../components/SessionList";
import { getModels, getSessions, logout } from "../api/client";

export function ChatPage() {
  const [models, setModels] = useState<{ model_id: string; remaining_tokens: number }[]>([]);
  const [selectedModel, setSelectedModel] = useState("");
  const [sessions, setSessions] = useState<string[]>([]);
  const [currentSession, setCurrentSession] = useState(uuidv4());
  const [sessionHasMessages, setSessionHasMessages] = useState(false);

  useEffect(() => {
    getModels().then((data) => {
      setModels(data.models ?? []);
      if (data.models?.length) setSelectedModel(data.models[0].model_id);
    }).catch(() => {});
    getSessions().then((data) => setSessions(data.sessions ?? [])).catch(() => {});
  }, []);

  function newSession() {
    const id = uuidv4();
    setCurrentSession(id);
    setSessions((prev) => [id, ...prev]);
    setSessionHasMessages(false);
  }

  function handleModelChange(model: string) {
    setSelectedModel(model);
    if (sessionHasMessages) newSession();
  }

  return (
    <div className="flex h-screen">
      <SessionList
        sessions={sessions}
        current={currentSession}
        onSelect={setCurrentSession}
        onNew={newSession}
      />
      <div className="flex-1 flex flex-col min-w-0">
        <div className="border-b px-4 py-2 flex items-center gap-3">
          <span className="text-sm text-gray-500">模型：</span>
          <ModelSelector models={models} selected={selectedModel} onChange={handleModelChange} />
          <div className="ml-auto">
            <button
              onClick={logout}
              className="text-sm text-gray-400 hover:text-gray-600 px-2 py-1"
            >
              退出登录
            </button>
          </div>
        </div>
        <ChatWindow model={selectedModel} sessionId={currentSession} onModelChange={setSelectedModel} onHasMessages={setSessionHasMessages} />
      </div>
    </div>
  );
}
