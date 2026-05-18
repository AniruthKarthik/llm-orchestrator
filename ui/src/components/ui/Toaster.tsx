import React, { useEffect, useState } from 'react';
import { useToastStore, type ToastType } from '@/store/useToastStore';
import { CheckCircle2, XCircle, Info, AlertTriangle, X } from 'lucide-react';

const icons: Record<ToastType, React.ReactNode> = {
  success: <CheckCircle2 size={16} />,
  error: <XCircle size={16} />,
  info: <Info size={16} />,
  warning: <AlertTriangle size={16} />,
};

const colorMap: Record<ToastType, { bg: string; border: string; icon: string; text: string }> = {
  success: { bg: '#f0fdf4', border: '#bbf7d0', icon: '#16a34a', text: '#15803d' },
  error:   { bg: '#fef2f2', border: '#fecaca', icon: '#dc2626', text: '#b91c1c' },
  info:    { bg: '#eff6ff', border: '#bfdbfe', icon: '#2563eb', text: '#1d4ed8' },
  warning: { bg: '#fffbeb', border: '#fde68a', icon: '#d97706', text: '#b45309' },
};

function ToastItem({ toast }: { toast: ReturnType<typeof useToastStore.getState>['toasts'][number] }) {
  const { remove } = useToastStore();
  const [visible, setVisible] = useState(false);
  const colors = colorMap[toast.type];

  useEffect(() => {
    // Trigger enter animation
    const t = requestAnimationFrame(() => setVisible(true));
    return () => cancelAnimationFrame(t);
  }, []);

  return (
    <div
      role="alert"
      aria-live="assertive"
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: '10px',
        padding: '12px 16px',
        borderRadius: '8px',
        border: `1px solid ${colors.border}`,
        background: colors.bg,
        boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
        pointerEvents: 'auto',
        minWidth: '280px',
        maxWidth: '380px',
        transition: 'opacity 250ms ease, transform 250ms ease',
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateX(0)' : 'translateX(24px)',
      }}
    >
      <span style={{ color: colors.icon, flexShrink: 0, marginTop: '1px' }}>
        {icons[toast.type]}
      </span>
      <span style={{
        fontSize: '13px',
        fontWeight: 500,
        color: colors.text,
        flex: 1,
        lineHeight: '1.4',
      }}>
        {toast.message}
      </span>
      <button
        onClick={() => remove(toast.id)}
        style={{
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: colors.icon,
          opacity: 0.6,
          padding: '0',
          flexShrink: 0,
          marginTop: '1px',
        }}
        aria-label="Dismiss"
      >
        <X size={14} />
      </button>
    </div>
  );
}

export function Toaster() {
  const { toasts } = useToastStore();

  if (toasts.length === 0) return null;

  return (
    <div
      style={{
        position: 'fixed',
        bottom: '20px',
        right: '20px',
        zIndex: 9999,
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
        pointerEvents: 'none',
      }}
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} />
      ))}
    </div>
  );
}
