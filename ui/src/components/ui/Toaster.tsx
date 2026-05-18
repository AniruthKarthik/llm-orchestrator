import { useToastStore, type ToastType } from '@/store/useToastStore';
import { CheckCircle2, XCircle, Info, AlertTriangle, X } from 'lucide-react';
import { cn } from '@/lib/utils';

const icons: Record<ToastType, React.ReactNode> = {
  success: <CheckCircle2 size={16} className="text-green-500 shrink-0" />,
  error: <XCircle size={16} className="text-red-500 shrink-0" />,
  info: <Info size={16} className="text-blue-500 shrink-0" />,
  warning: <AlertTriangle size={16} className="text-amber-500 shrink-0" />,
};

const styles: Record<ToastType, string> = {
  success: 'border-green-500/20 bg-green-500/10 text-green-900 dark:text-green-100',
  error: 'border-red-500/20 bg-red-500/10 text-red-900 dark:text-red-100',
  info: 'border-blue-500/20 bg-blue-500/10 text-blue-900 dark:text-blue-100',
  warning: 'border-amber-500/20 bg-amber-500/10 text-amber-900 dark:text-amber-100',
};

export function Toaster() {
  const { toasts, remove } = useToastStore();

  if (toasts.length === 0) return null;

  return (
    <div
      aria-live="polite"
      aria-atomic="false"
      className="fixed bottom-4 right-4 z-[9999] flex flex-col gap-2 max-w-sm w-full pointer-events-none"
    >
      {toasts.map((toast) => (
        <div
          key={toast.id}
          role="alert"
          className={cn(
            'flex items-start gap-3 px-4 py-3 rounded-lg border shadow-lg pointer-events-auto',
            'animate-in slide-in-from-right-4 fade-in-0 duration-300',
            styles[toast.type]
          )}
        >
          {icons[toast.type]}
          <span className="text-sm font-medium flex-1 leading-snug">{toast.message}</span>
          <button
            onClick={() => remove(toast.id)}
            className="text-current opacity-50 hover:opacity-100 transition-opacity shrink-0 mt-0.5"
            aria-label="Dismiss"
          >
            <X size={14} />
          </button>
        </div>
      ))}
    </div>
  );
}
