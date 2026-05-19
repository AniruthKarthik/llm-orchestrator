import { useState } from 'react';
import { AlertTriangle, X } from 'lucide-react';

export function InMemoryBanner() {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;

  return (
    <div
      role="alert"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        padding: '10px 16px',
        background: 'linear-gradient(90deg, #78350f11 0%, #92400e11 100%)',
        borderBottom: '1px solid #d9770640',
        fontSize: '12px',
        fontWeight: 500,
        color: '#92400e',
        flexShrink: 0,
      }}
    >
      <AlertTriangle size={14} style={{ flexShrink: 0, color: '#d97706' }} />
      <span style={{ flex: 1 }}>
        <strong>No database connected</strong> — running in in-memory mode.
        All workflows, task results, artifacts, and agents will be permanently lost when the server restarts or the page is refreshed.
        Set <code style={{ background: '#d9770620', padding: '0 4px', borderRadius: 3, fontFamily: 'monospace' }}>DATABASE_URL</code> to persist data.
      </span>
      <button
        onClick={() => setDismissed(true)}
        style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#92400e', opacity: 0.7, padding: 2 }}
        aria-label="Dismiss"
      >
        <X size={14} />
      </button>
    </div>
  );
}
