import { createContext, useContext } from 'react';
import type { WsEvent } from '@/hooks/useWebSocket';

interface WsContextValue {
  isConnected: boolean;
  events: WsEvent[];
  addListener: (fn: (e: WsEvent) => void) => () => void;
}

export const WsContext = createContext<WsContextValue>({
  isConnected: false,
  events: [],
  addListener: () => () => {},
});

export const useWsContext = () => useContext(WsContext);
