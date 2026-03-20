import { useState, useCallback } from "react";
import { chatStream } from "../api/client";

export function useStream(model: string, sessionId: string) {
  const [streaming, setStreaming] = useState(false);
  const [buffer, setBuffer] = useState("");

  const send = useCallback(
    (messages: { role: string; content: string }[]) => {
      setStreaming(true);
      setBuffer("");
      chatStream(
        model,
        messages,
        sessionId,
        (chunk) => setBuffer((prev) => prev + chunk),
        () => setStreaming(false),
      );
    },
    [model, sessionId],
  );

  return { send, buffer, streaming };
}
