import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../api/client";
import { stripAnsi } from "../utils/text";

const MAX_LINES = 500;

interface UseAgentOutputResult {
  lines: string[];
  isConnected: boolean;
  isPaused: boolean;
  togglePause: () => void;
}

/**
 * Streams an agent's terminal output via peek + SSE.
 *
 * Buffers up to 500 lines. Supports pause/resume to freeze the display
 * while still buffering incoming data.
 */
export function useAgentOutput(agentName: string): UseAgentOutputResult {
  const [lines, setLines] = useState<string[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isPaused, setIsPaused] = useState(false);

  // Buffer holds lines while paused so we don't lose data
  const bufferRef = useRef<string[]>([]);
  const isPausedRef = useRef(false);

  const togglePause = useCallback(() => {
    setIsPaused((prev) => {
      const next = !prev;
      isPausedRef.current = next;
      if (!next) {
        // Resuming — flush buffer to display
        setLines(bufferRef.current.slice(-MAX_LINES));
      }
      return next;
    });
  }, []);

  const appendLines = useCallback((newLines: string[]) => {
    bufferRef.current = [...bufferRef.current, ...newLines].slice(-MAX_LINES);
    if (!isPausedRef.current) {
      setLines(bufferRef.current);
    }
  }, []);

  // Fetch initial peek output
  useEffect(() => {
    bufferRef.current = [];
    setLines([]);
    setIsConnected(false);
    setIsPaused(false);
    isPausedRef.current = false;

    api
      .getAgentPeek(agentName, 100)
      .then(({ output }) => {
        if (output) {
          const initial = stripAnsi(output).split("\n");
          bufferRef.current = initial.slice(-MAX_LINES);
          setLines(bufferRef.current);
        }
      })
      .catch(() => {
        // Peek may fail for stopped agents
      });
  }, [agentName]);

  // Connect SSE stream
  useEffect(() => {
    const es = new EventSource(
      `/api/agents/${encodeURIComponent(agentName)}/output`,
    );
    let errorCount = 0;

    const handleOutput = (e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data as string) as { output?: string };
        if (parsed.output) {
          const incoming = stripAnsi(parsed.output).split("\n");
          appendLines(incoming);
        }
      } catch {
        // ignore malformed data
      }
    };

    es.onopen = () => {
      setIsConnected(true);
      errorCount = 0;
    };

    es.onmessage = handleOutput;
    es.addEventListener("agent.output", handleOutput as EventListener);

    es.onerror = () => {
      errorCount++;
      if (errorCount > 3) {
        setIsConnected(false);
        es.close();
      }
    };

    return () => {
      es.close();
      setIsConnected(false);
    };
  }, [agentName, appendLines]);

  return { lines, isConnected, isPaused, togglePause };
}
