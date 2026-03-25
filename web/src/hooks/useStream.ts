import { useState, useCallback } from "react";
import { chatStream } from "../api/client";

export function useStream(model: string, sessionId: string) {
  const [streaming, setStreaming] = useState(false);
  const [buffer, setBuffer] = useState("");

  const send = useCallback(
    (messages: { role: string; content: string }[], onDone?: (content: string) => void) => {
      setStreaming(true);
      setBuffer("");

      let accumulated = "";
      let partial = ""; // incomplete SSE line buffer

      chatStream(
        model,
        messages,
        sessionId,
        (rawChunk) => {
          // Non-SSE error from client.ts error handling
          if (rawChunk.startsWith("[ERROR]")) {
            accumulated = rawChunk;
            setBuffer(accumulated);
            return;
          }
          // SSE format from gin: "data: <text>\n\n"
          // rawChunk may span multiple events or be split mid-line
          partial += rawChunk;
          const parts = partial.split("\n\n");
          partial = parts.pop() ?? ""; // keep the potentially incomplete tail

          for (const part of parts) {
            // Collect all data: lines in one SSE event and join with \n
            // (Gin splits multi-line text across multiple data: fields)
            const dataLines: string[] = [];
            let done = false;
            for (const line of part.split("\n")) {
              if (!line.startsWith("data: ")) continue;
              const data = line.slice(6);
              if (data === "[DONE]") { done = true; break; }
              dataLines.push(data);
            }
            if (dataLines.length > 0) {
              accumulated += dataLines.join("\n");
              setBuffer(accumulated);
            }
            if (done) return;
          }
        },
        () => {
          setStreaming(false);
          onDone?.(accumulated);
        },
      );
    },
    [model, sessionId],
  );

  return { send, buffer, streaming };
}
