import { useState } from 'react';
import { useWebSocket } from '@/hooks/useWebSocket';
import { Activity, CheckCircle2, XCircle, Clock, ShieldCheck, PlayCircle, Search, Filter, Terminal, AlertTriangle } from 'lucide-react';
import api from '@/api/client';
import { cn } from '@/lib/utils';

export default function ExecutionsPage() {
  const { events, isConnected } = useWebSocket();
  const [approving, setApproving] = useState<string | null>(null);
  const [filter, setFilter] = useState('');

  const handleApprove = async (workflowId: string, taskId: string) => {
    setApproving(taskId);
    try {
      await api.post(`/workflows/${workflowId}/tasks/${taskId}/approve`);
    } catch (error) {
      console.error('Approval failed', error);
    } finally {
      setApproving(null);
    }
  };

  const getEventIcon = (type: string) => {
    if (type.includes('Started')) return <PlayCircle size={14} className="text-blue-500" />;
    if (type.includes('Completed')) return <CheckCircle2 size={14} className="text-green-500" />;
    if (type.includes('Failed')) return <XCircle size={14} className="text-red-500" />;
    if (type.includes('Approval')) return <AlertTriangle size={14} className="text-amber-500" />;
    return <Activity size={14} className="text-muted-foreground" />;
  };

  const getEventColor = (type: string) => {
    if (type.includes('Started')) return 'border-blue-500/20 bg-blue-500/5';
    if (type.includes('Completed')) return 'border-green-500/20 bg-green-500/5';
    if (type.includes('Failed')) return 'border-red-500/20 bg-red-500/5';
    if (type.includes('Approval')) return 'border-amber-500/20 bg-amber-500/5';
    return 'border-border bg-transparent';
  };

  const reversedEvents = [...events].reverse().filter(e => e.type.toLowerCase().includes(filter.toLowerCase()));

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-4">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Execution Monitor</h1>
          <p className="text-sm text-muted-foreground">Real-time log streaming and workflow orchestration trace.</p>
        </div>
        <div className="flex items-center gap-3">
          <div className={cn("flex items-center gap-2 px-2.5 py-1 rounded border text-xs font-semibold", isConnected ? "bg-green-500/10 border-green-500/20 text-green-600" : "bg-red-500/10 border-red-500/20 text-red-600")}>
            <div className={cn("w-1.5 h-1.5 rounded-full", isConnected ? "bg-green-500 animate-pulse" : "bg-red-500")} />
            {isConnected ? 'LIVE STREAM' : 'DISCONNECTED'}
          </div>
        </div>
      </div>

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
                <button className="h-6 px-2 text-xs font-medium border border-border rounded bg-background hover:bg-secondary text-muted-foreground transition-colors flex items-center gap-1.5">
                  <Filter size={12} /> Filter
                </button>
             </div>
          </div>
          
          <div className="flex-1 overflow-y-auto font-mono text-xs bg-foreground text-background">
            {reversedEvents.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground flex flex-col items-center gap-2 h-full justify-center opacity-50">
                <Clock size={24} />
                <span>Waiting for execution events...</span>
              </div>
            ) : (
              <div className="divide-y divide-border/20">
                {reversedEvents.map((event, i) => (
                  <div key={i} className={cn("p-2 hover:bg-secondary/10 transition-colors border-l-2", getEventColor(event.type))}>
                    <div className="flex items-start gap-3">
                      <div className="w-24 shrink-0 text-muted-foreground">
                        {new Date(event.timestamp).toISOString().split('T')[1].slice(0, -1)}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          {getEventIcon(event.type)}
                          <span className={cn("font-bold", event.type.includes('Failed') ? 'text-red-400' : 'text-blue-300')}>{event.type}</span>
                        </div>
                        <div className="text-muted-foreground break-all bg-background/5 p-1.5 rounded">
                           {JSON.stringify(event.payload)}
                        </div>
                        
                        {/* Approval Action */}
                        {event.type === 'TaskWaitingForApproval' && (
                          <div className="mt-2 p-2 bg-amber-500/10 border border-amber-500/20 rounded flex items-center justify-between text-amber-200">
                            <div className="flex items-center gap-2 font-medium">
                              <ShieldCheck size={14} />
                              Manual execution approval required
                            </div>
                            <button 
                              onClick={() => handleApprove(event.payload.WorkflowID, event.payload.TaskID)}
                              disabled={approving === event.payload.TaskID}
                              className="bg-amber-500 text-amber-950 px-2 py-1 rounded font-bold hover:bg-amber-400 transition-colors disabled:opacity-50"
                            >
                              {approving === event.payload.TaskID ? 'Approving...' : 'Approve'}
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Sidebar Summary */}
        <div className="space-y-4">
           <div className="bg-card border border-border rounded-lg shadow-sm p-4">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
                <Activity size={14} /> Metric Stream
              </h3>
              <div className="space-y-4">
                 <div>
                   <div className="text-xs text-muted-foreground mb-1">Token Throughput (Last 5m)</div>
                   <div className="text-2xl font-bold text-foreground">1,204 / s</div>
                 </div>
                 <div>
                   <div className="text-xs text-muted-foreground mb-1">Active Workers</div>
                   <div className="text-xl font-bold text-foreground">12</div>
                 </div>
                 <div>
                   <div className="text-xs text-muted-foreground mb-1">Queue Depth</div>
                   <div className="text-xl font-bold text-orange-500">0</div>
                 </div>
              </div>
           </div>

           <div className="bg-card border border-border rounded-lg shadow-sm p-4">
              <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
                <CheckCircle2 size={14} /> Systems
              </h4>
              <div className="space-y-2">
                 {[
                   { name: 'Redis Broker', status: 'OK' },
                   { name: 'Postgres State', status: 'OK' },
                   { name: 'Provider: Groq', status: 'OK' },
                   { name: 'Provider: Anthropic', status: 'OK' },
                 ].map(sys => (
                   <div key={sys.name} className="flex items-center justify-between text-xs">
                      <span className="text-muted-foreground">{sys.name}</span>
                      <span className="font-semibold text-green-600">{sys.status}</span>
                   </div>
                 ))}
              </div>
           </div>
        </div>
      </div>
    </div>
  );
}
