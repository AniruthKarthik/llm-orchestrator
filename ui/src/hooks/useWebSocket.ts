import { useEffect, useRef, useState, useCallback } from 'react';

export interface WsEvent {
  id?: string;
  type: string;
  workflowId: string;
  taskId?: string;
  timestamp: string;
  payload?: Record<string, unknown>;
}

const WS_RECONNECT_DELAY_MS = 3000;
const MAX_EVENTS = 200;

export function useWebSocket() {
  const [events, setEvents] = useState<WsEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isMounted = useRef(true);
  // callbacks registered by consumers (e.g. pages that want live updates)
  const listenersRef = useRef<Array<(e: WsEvent) => void>>([]);

  const addListener = useCallback((fn: (e: WsEvent) => void) => {
    listenersRef.current.push(fn);
    return () => {
      listenersRef.current = listenersRef.current.filter((l) => l !== fn);
    };
  }, []);

  useEffect(() => {
    isMounted.current = true;

    function connect() {
      if (!isMounted.current) return;
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return;

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = import.meta.env.VITE_WS_URL || `${window.location.host}/api/v1/ws`;
      const cleanHost = host.replace(/^wss?:\/\//, '');
      const url = `${protocol}//${cleanHost}`;

      const socket = new WebSocket(url);
      wsRef.current = socket;

      socket.onopen = () => {
        if (!isMounted.current) { socket.close(); return; }
        setIsConnected(true);
      };

      socket.onmessage = (msg) => {
        try {
          const data = JSON.parse(msg.data) as WsEvent;
          setEvents((prev) => {
            const next = [...prev, data];
            return next.length > MAX_EVENTS ? next.slice(next.length - MAX_EVENTS) : next;
          });
          // Notify all active page listeners
          listenersRef.current.forEach((fn) => fn(data));
        } catch (err) {
          console.warn('WebSocket: failed to parse message', err);
        }
      };

      socket.onclose = () => {
        setIsConnected(false);
        if (!isMounted.current) return;
        reconnectTimer.current = setTimeout(connect, WS_RECONNECT_DELAY_MS);
      };

      socket.onerror = () => { socket.close(); };
    }

    connect();

    return () => {
      isMounted.current = false;
      if (reconnectTimer.current) { clearTimeout(reconnectTimer.current); }
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, []);

  return { events, isConnected, addListener };
}
