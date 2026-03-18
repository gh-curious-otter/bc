import { useEffect, useRef, useState, useCallback } from 'react';
import type { WSEvent, WSEventType } from '../api/types';

type Listener = (event: WSEvent) => void;

export function useWebSocket() {
  const esRef = useRef<EventSource | null>(null);
  const listenersRef = useRef<Map<WSEventType, Set<Listener>>>(new Map());
  const [connected, setConnected] = useState(false);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();

  const connect = useCallback(() => {
    let es: EventSource;
    try {
      es = new EventSource('/api/events');
    } catch {
      // EventSource not available — degrade gracefully
      return;
    }

    es.onopen = () => {
      setConnected(true);
    };

    es.onmessage = (e: MessageEvent) => {
      try {
        const event = JSON.parse(e.data as string) as WSEvent;
        const listeners = listenersRef.current.get(event.type);
        listeners?.forEach((fn) => fn(event));
      } catch {
        // ignore malformed messages
      }
    };

    es.onerror = () => {
      setConnected(false);
      es.close();
      reconnectTimer.current = setTimeout(connect, 3000);
    };

    esRef.current = es;
  }, []);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      esRef.current?.close();
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
