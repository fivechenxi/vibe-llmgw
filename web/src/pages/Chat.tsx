import { useEffect, useState } from "react";
import { v4 as uuidv4 } from "uuid";
import { ChatWindow } from "../components/ChatWindow";
import { ModelSelector } from "../components/ModelSelector";
import { SessionList } from "../components/SessionList";
import { getModels, getSessions } from "../api/client";

export function ChatPage() {
  const [models, setModels] = useState<{ model_id: string; remaining_tokens: number }[]>([]);
  const [selectedModel, setSelectedModel] = useState("");
  const [sessions, setSessions] = useState<string[]>([]);
  const [currentSession, setCurrentSession] = useState(uuidv4());

  useEffect(() => {
    getModels().then((data) => {
      setModels(data.models ?? []);
      if (data.models?.length) setSelectedModel(data.models[0].model_id);
    });
    getSessions().then((data) => setSessions(data.sessions ?? []));
  }, []);

  function newSession() {
    const id = uuidv4();
    setCurrentSession(id);
    setSessions((prev) => [id, ...prev]);
  }

  return (
    <div className="flex h-screen">
      <SessionList
        sessions={sessions}
        current={currentSession}
        onSelect={setCurrentSession}
        onNew={newSession}
      />
      <div className="flex-1 flex flex-col">
        <div className="border-b px-4 py-2 flex items-center gap-3">
          <span className="text-sm text-gray-500">模型：</span>
          <ModelSelector models={models} selected={selectedModel} onChange={setSelectedModel} />
        </div>
        <ChatWindow model={selectedModel} sessionId={currentSession} />
      </div>
    </div>
  );
}
