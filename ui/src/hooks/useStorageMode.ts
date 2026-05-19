import { useEffect, useState } from 'react';
import api from '@/api/client';

export type StorageMode = 'memory' | 'postgres' | null;

let cachedMode: StorageMode = null;

export function useStorageMode(): StorageMode {
  const [mode, setMode] = useState<StorageMode>(cachedMode);

  useEffect(() => {
    if (cachedMode !== null) {
      setMode(cachedMode);
      return;
    }
    api.get('/health').then((res) => {
      const m: StorageMode = res.data?.storageMode === 'postgres' ? 'postgres' : 'memory';
      cachedMode = m;
      setMode(m);
    }).catch(() => {
      setMode('memory'); // assume worst case
    });
  }, []);

  return mode;
}
