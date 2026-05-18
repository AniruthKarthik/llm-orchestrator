import { useEffect, useRef, useState } from 'react';
import type { Event } from '@/types';

const WS_RECONNECT_DELAY_MS = 5000;
const MAX_EVENTS = 500; // Prevent unbounded memory growth

export function useWebSocket() {
  const [events, setEvents] = useState<Event[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Tracks whether the hook is still mounted so we don't reconnect after unmount
  const isMounted = useRef(true);

  useEffect(() => {
    isMounted.current = true;

    function connect() {
      if (!isMounted.current) return;
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return;

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      // VITE_WS_URL should be just the host+path without protocol, e.g. "localhost:8080/api/v1/ws"
      const host = import.meta.env.VITE_WS_URL || 'localhost:8080/api/v1/ws';
      // Strip protocol prefix if the env var accidentally includes one
      const cleanHost = host.replace(/^wss?:\/\//, '');
      const url = `${protocol}//${cleanHost}`;

      const socket = new WebSocket(url);
      wsRef.current = socket;

      socket.onopen = () => {
        if (!isMounted.current) {
          socket.close();
          return;
        }
        setIsConnected(true);
      };

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as Event;
          setEvents((prev) => {
            const next = [...prev, data];
            return next.length > MAX_EVENTS ? next.slice(next.length - MAX_EVENTS) : next;
          });
        } catch (err) {
          console.warn('WebSocket: failed to parse message', err);
        }
      };

      socket.onclose = () => {
        setIsConnected(false);
        if (!isMounted.current) return;
        reconnectTimer.current = setTimeout(connect, WS_RECONNECT_DELAY_MS);
      };

      socket.onerror = () => {
        // onclose fires after onerror, which handles reconnect
        socket.close();
      };
    }

    connect();

    return () => {
      isMounted.current = false;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
      if (wsRef.current) {
        wsRef.current.onclose = null; // Prevent reconnect trigger on intentional close
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, []); // Intentionally empty — connect once on mount

  return { events, isConnected };
}
