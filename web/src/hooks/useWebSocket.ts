import { useEffect, useRef, useState, useCallback } from 'react';
import type { WSEvent, WSEventType } from '../api/types';

type Listener = (event: WSEvent) => void;

const BASE_DELAY = 1000;
const MAX_DELAY = 30000;

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const listenersRef = useRef<Map<WSEventType, Set<Listener>>>(new Map());
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();
  const retriesRef = useRef(0);

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    let ws: WebSocket;
    try {
      ws = new WebSocket(`${protocol}//${window.location.host}/api/events`);
    } catch {
      return;
    }

    ws.onopen = () => {
      setConnected(true);
      retriesRef.current = 0; // Reset backoff on successful connection
      ws.send(JSON.stringify({
        action: 'subscribe',
        types: ['agent.*', 'channel.message', 'cost.updated', 'cost.budget_alert'],
      }));
    };

    ws.onmessage = (e: MessageEvent) => {
      try {
        const event = JSON.parse(e.data as string) as WSEvent;
        const listeners = listenersRef.current.get(event.type);
        listeners?.forEach((fn) => fn(event));
      } catch {
        // ignore malformed messages
      }
    };

    ws.onclose = () => {
      setConnected(false);
      // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
      const delay = Math.min(BASE_DELAY * Math.pow(2, retriesRef.current), MAX_DELAY);
      retriesRef.current++;
      reconnectTimer.current = setTimeout(connect, delay);
    };

    ws.onerror = () => {
      ws.close();
    };

    wsRef.current = ws;
  }, []);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const subscribe = useCallback((type: WSEventType, listener: Listener) => {
    if (!listenersRef.current.has(type)) {
      listenersRef.current.set(type, new Set());
    }
    listenersRef.current.get(type)!.add(listener);
    return () => {
      listenersRef.current.get(type)?.delete(listener);
    };
  }, []);

  return { connected, subscribe };
}
