import { useWsContext } from '@/context/WsContext';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { useEffect, useState } from 'react';
import {
  Activity,
  CheckCircle2,
  XCircle,
  Clock,
  ShieldCheck,
  PlayCircle,
  Search,
  Terminal,
  AlertTriangle,
  Loader2,
} from 'lucide-react';
import api from '@/api/client';
import { cn } from '@/lib/utils';
import type { WsEvent } from '@/hooks/useWebSocket';

export default function ExecutionsPage() {
  const { events, isConnected } = useWsContext();
  const { providers, fetchProviders } = useWorkflowStore();
  const [approving, setApproving] = useState<string | null>(null);
  const [approvalError, setApprovalError] = useState<string | null>(null);
  const [filter, setFilter] = useState('');
  const [historyEvents, setHistoryEvents] = useState<WsEvent[]>([]);

  useEffect(() => {
    fetchProviders();
    api.get('/events')
      .then((response) => setHistoryEvents(response.data ?? []))
      .catch(() => setHistoryEvents([]));
  }, [fetchProviders]);

  const handleApprove = async (workflowId: string, taskId: string) => {
    setApproving(taskId);
    setApprovalError(null);
    try {
      await api.post(`/workflows/${workflowId}/tasks/${taskId}/approve`);
    } catch (error: unknown) {
      let msg = 'Approval failed. Task may have already been processed.';
      const err = error as Record<string, unknown>;
      if (err.response && typeof err.response === 'object') {
        const resp = err.response as Record<string, unknown>;
        if (resp.data && typeof resp.data === 'object') {
          const data = resp.data as Record<string, string>;
          if (data.error) msg = data.error;
        }
      }
      setApprovalError(msg);
    } finally {
      setApproving(null);
    }
  };

  const getEventIcon = (type: string) => {
    if (type.includes('STARTED')) return <PlayCircle size={14} className="text-blue-500" />;
    if (type.includes('COMPLETED')) return <CheckCircle2 size={14} className="text-green-500" />;
    if (type.includes('FAILED')) return <XCircle size={14} className="text-red-500" />;
    if (type.includes('APPROVAL') || type.includes('WAITING')) return <AlertTriangle size={14} className="text-amber-500" />;
    return <Activity size={14} className="text-muted-foreground" />;
  };

  const getEventColor = (type: string) => {
    if (type.includes('STARTED')) return 'border-blue-500/20 bg-blue-500/5';
    if (type.includes('COMPLETED')) return 'border-green-500/20 bg-green-500/5';
    if (type.includes('FAILED')) return 'border-red-500/20 bg-red-500/5';
    if (type.includes('APPROVAL') || type.includes('WAITING')) return 'border-amber-500/20 bg-amber-500/5';
    return 'border-border bg-transparent';
  };

  const eventMap = new Map<string, WsEvent>();
  [...historyEvents, ...events].forEach((event) => {
    const key = event.id ?? `${event.timestamp}-${event.type}-${event.workflowId}-${event.taskId ?? ''}`;
    eventMap.set(key, event);
  });
  const combinedEvents = Array.from(eventMap.values());

  const reversedEvents = [...combinedEvents]
    .reverse()
    .filter(
      (e) =>
        e.type.toLowerCase().includes(filter.toLowerCase()) ||
        JSON.stringify(e.payload).toLowerCase().includes(filter.toLowerCase())
    );

  const getPayload = (payload: unknown): Record<string, unknown> => {
    if (!payload || typeof payload !== 'object') return {};
    return payload as Record<string, unknown>;
  };

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-4">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Execution Monitor</h1>
          <p className="text-sm text-muted-foreground">
            Real-time log streaming and workflow orchestration trace.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'flex items-center gap-2 px-2.5 py-1 rounded border text-xs font-semibold',
              isConnected
                ? 'bg-green-500/10 border-green-500/20 text-green-600'
                : 'bg-red-500/10 border-red-500/20 text-red-600'
            )}
          >
            <div
              className={cn(
                'w-1.5 h-1.5 rounded-full',
                isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'
              )}
            />
            {isConnected ? 'LIVE STREAM' : 'DISCONNECTED'}
          </div>
        </div>
      </div>

      {approvalError && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-600 px-4 py-2 rounded text-xs flex items-center gap-2">
          <XCircle size={14} />
          {approvalError}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
        {/* Main Logs Area */}
        <div className="lg:col-span-3 bg-card border border-border rounded-lg shadow-sm flex flex-col h-[700px]">
          <div className="h-10 border-b border-border flex items-center px-3 gap-2 bg-muted/30 shrink-0">
            <Terminal size={14} className="text-muted-foreground" />
            <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Event Trace</span>
            <div className="ml-auto flex items-center gap-2">
              <div className="relative">
                <Search size={12} className="absolute left-2 top-1.5 text-muted-foreground" />
                <input
                  type="text"
                  placeholder="Filter events..."
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                  className="h-6 bg-background border border-border rounded text-xs pl-6 pr-2 focus:outline-none focus:ring-1 focus:ring-primary w-48"
                />
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto font-mono text-xs bg-[#0d1117] text-[#e6edf3]">
            {reversedEvents.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground flex flex-col items-center gap-2 h-full justify-center opacity-50">
                <Clock size={24} />
                <span>
                  {isConnected
                    ? 'Waiting for execution events...'
                    : 'WebSocket disconnected — reconnecting...'}
                </span>
              </div>
            ) : (
              <div className="divide-y divide-white/5">
                {reversedEvents.map((event, i) => {
                  const payload = getPayload(event.payload);
                  return (
                    <div key={i} className={cn('p-2 hover:bg-white/5 transition-colors border-l-2', getEventColor(event.type))}>
                      <div className="flex items-start gap-3">
                        <div className="w-24 shrink-0 text-muted-foreground text-[10px]">
                          {new Date(event.timestamp).toLocaleTimeString()}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            {getEventIcon(event.type)}
                            <span
                              className={cn(
                                'font-bold uppercase tracking-tight',
                                event.type.includes('FAILED') ? 'text-red-400' : 'text-blue-400'
                              )}
                            >
                              {event.type}
                            </span>
                          </div>
                          <div className="text-[#8b949e] break-all p-1 rounded font-mono text-[11px]">
                            {JSON.stringify(payload, null, 2)}
                          </div>

                          {/* Approval Action */}
                          {event.type === 'TASK_WAITING_FOR_APPROVAL' && (
                            <div className="mt-2 p-2 bg-amber-500/10 border border-amber-500/20 rounded flex items-center justify-between text-amber-200">
                              <div className="flex items-center gap-2 font-medium">
                                <ShieldCheck size={14} />
                                Manual execution approval required
                              </div>
                              <button
                                onClick={() =>
                                  handleApprove(
                                    event.workflowId,
                                    event.taskId ?? ''
                                  )
                                }
                                disabled={!event.taskId || approving === event.taskId}
                                className="bg-amber-500 text-amber-950 px-2 py-1 rounded font-bold hover:bg-amber-400 transition-colors disabled:opacity-50 flex items-center gap-1.5"
                              >
                                {approving === event.taskId ? (
                                  <><Loader2 size={12} className="animate-spin" /> Approving...</>
                                ) : (
                                  'Approve'
                                )}
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* Sidebar Summary */}
        <div className="space-y-4">
          <div className="bg-card border border-border rounded-lg shadow-sm p-4">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
              <Activity size={14} /> Real-time Feed
            </h3>
            <div className="space-y-4">
              <div>
                <div className="text-xs text-muted-foreground mb-1">WebSocket Status</div>
                <div className={cn('text-xs font-bold', isConnected ? 'text-green-600' : 'text-red-600')}>
                  {isConnected ? 'CONNECTED' : 'DISCONNECTED'}
                </div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground mb-1">Total Session Events</div>
                <div className="text-xl font-bold text-foreground">{combinedEvents.length}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground mb-1">Filtered</div>
                <div className="text-xl font-bold text-foreground">{reversedEvents.length}</div>
              </div>
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg shadow-sm p-4">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
              <CheckCircle2 size={14} /> Configured Providers
            </h4>
            <div className="space-y-2">
              {providers.isLoading ? (
                <div className="flex justify-center py-2">
                  <Loader2 size={14} className="animate-spin text-muted-foreground" />
                </div>
              ) : providers.data.length === 0 ? (
                <div className="text-xs text-muted-foreground italic">No providers configured</div>
              ) : (
                providers.data.map((p) => (
                  <div key={p.name} className="flex items-center justify-between text-xs">
                    <span className="text-muted-foreground capitalize">{p.name}</span>
                    <span className="font-semibold text-secondary-foreground bg-secondary px-1.5 py-0.5 rounded text-[10px]">
                      {p.models.length} models
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
