import { useEffect, useRef, useCallback } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";

interface WebTerminalProps {
  agentName: string;
  onDisconnect?: () => void;
}

export function WebTerminal({ agentName, onDisconnect }: WebTerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  const connect = useCallback(() => {
    if (!containerRef.current) return;

    // Create terminal
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: "'Space Mono', 'Menlo', 'Consolas', monospace",
      theme: {
        background: "#0a0a0f",
        foreground: "#e0e0e0",
        cursor: "#e0e0e0",
        selectionBackground: "#3a3a5a",
      },
    });
    termRef.current = term;

    const fitAddon = new FitAddon();
    fitRef.current = fitAddon;
    term.loadAddon(fitAddon);
    term.loadAddon(new WebLinksAddon());

    term.open(containerRef.current);
    fitAddon.fit();

    // Connect WebSocket
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${proto}//${window.location.host}/api/agents/${encodeURIComponent(agentName)}/terminal`;
    const ws = new WebSocket(wsUrl);
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    ws.onopen = () => {
      // Send initial size
      const dims = fitAddon.proposeDimensions();
      if (dims) {
        ws.send(JSON.stringify({ type: "resize", cols: dims.cols, rows: dims.rows }));
      }
    };

    ws.onmessage = (evt: MessageEvent) => {
      if (evt.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(evt.data));
      } else {
        term.write(evt.data as string);
      }
    };

    ws.onclose = () => {
      term.write("\r\n\x1b[90m[terminal disconnected]\x1b[0m\r\n");
      onDisconnect?.();
    };

    ws.onerror = () => {
      term.write("\r\n\x1b[31m[connection error]\x1b[0m\r\n");
    };

    // Terminal input → WebSocket
    term.onData((data: string) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // Terminal binary input → WebSocket
    term.onBinary((data: string) => {
      if (ws.readyState === WebSocket.OPEN) {
        const buf = new Uint8Array(data.length);
        for (let i = 0; i < data.length; i++) {
          buf[i] = data.charCodeAt(i);
        }
        ws.send(buf.buffer);
      }
    });

    // Handle resize
    const onResize = () => {
      fitAddon.fit();
      const dims = fitAddon.proposeDimensions();
      if (dims && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: dims.cols, rows: dims.rows }));
      }
    };

    term.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols, rows }));
      }
    });

    const resizeObserver = new ResizeObserver(onResize);
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      ws.close();
      term.dispose();
    };
  }, [agentName, onDisconnect]);

  useEffect(() => {
    const cleanup = connect();
    return () => cleanup?.();
  }, [connect]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full min-h-[300px] rounded-lg border border-bc-border/50 overflow-hidden"
    />
  );
}
