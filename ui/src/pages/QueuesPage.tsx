import { useEffect, useState } from 'react';
import { Search, ListTodo, Loader2, RefreshCw } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { cn } from '@/lib/utils';

export default function QueuesPage() {
  const { queueTasks, fetchQueues } = useWorkflowStore();
  const [search, setSearch] = useState('');

  useEffect(() => {
    fetchQueues();
    const interval = setInterval(fetchQueues, 5000);
    return () => clearInterval(interval);
  }, [fetchQueues]);

  const filtered = queueTasks.data.filter(
    (t) =>
      t.id?.toLowerCase().includes(search.toLowerCase()) ||
      t.workflowId?.toLowerCase().includes(search.toLowerCase()) ||
      t.name?.toLowerCase().includes(search.toLowerCase())
  );

  const statusCounts = {
    PENDING: queueTasks.data.filter((t) => t.status === 'PENDING').length,
    RUNNING: queueTasks.data.filter((t) => t.status === 'RUNNING').length,
    WAITING: queueTasks.data.filter((t) => t.status === 'WAITING_FOR_APPROVAL').length,
  };

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-foreground">Queue Management</h1>
          <p className="text-sm text-muted-foreground">
            Monitor pending, running, and approval-gated tasks. Refreshes every 5s.
          </p>
        </div>
        <button
          onClick={fetchQueues}
          disabled={queueTasks.isLoading}
          className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground border border-border px-3 py-1.5 rounded-md hover:bg-secondary transition-colors disabled:opacity-50"
        >
          <RefreshCw size={13} className={cn(queueTasks.isLoading && 'animate-spin')} />
          Refresh
        </button>
      </div>

      {queueTasks.error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-3 rounded-lg text-xs">
          {queueTasks.error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {[
          { label: 'Pending', count: statusCounts.PENDING, color: 'text-orange-600' },
          { label: 'Running', count: statusCounts.RUNNING, color: 'text-blue-600' },
          { label: 'Awaiting Approval', count: statusCounts.WAITING, color: 'text-amber-600' },
        ].map((s) => (
          <div key={s.label} className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="text-[10px] font-bold text-muted-foreground uppercase mb-1">{s.label}</div>
            <div className={cn('text-2xl font-bold', s.color)}>{s.count}</div>
          </div>
        ))}
      </div>

      <div className="bg-card border border-border rounded-lg shadow-sm">
        <div className="p-3 border-b border-border flex items-center justify-between bg-muted/20">
          <span className="text-xs font-bold text-muted-foreground">
            Global Task Queue ({filtered.length})
          </span>
          <div className="relative">
            <Search size={12} className="absolute left-2.5 top-2 text-muted-foreground" />
            <input
              className="h-7 w-48 bg-background border border-border rounded text-[11px] pl-8 pr-2 focus:ring-1 focus:ring-primary outline-none"
              placeholder="Search by ID, workflow, or name..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>
        <div className="overflow-x-auto">
          {queueTasks.isLoading && queueTasks.data.length === 0 ? (
            <div className="py-20 flex justify-center">
              <Loader2 size={24} className="animate-spin text-muted-foreground" />
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-20 text-center">
              <ListTodo size={48} className="mx-auto text-muted-foreground opacity-20 mb-4" />
              <p className="text-muted-foreground text-sm">
                {search ? 'No tasks match your search.' : 'The task queue is currently empty.'}
              </p>
            </div>
          ) : (
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="bg-muted/30 border-b border-border text-muted-foreground font-bold uppercase tracking-tighter">
                  <th className="px-4 py-2.5">Task ID</th>
                  <th className="px-4 py-2.5">Name</th>
                  <th className="px-4 py-2.5">Workflow</th>
                  <th className="px-4 py-2.5">Status</th>
                  <th className="px-4 py-2.5">Created At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border font-medium">
                {filtered.map((task) => (
                  <tr key={task.id} className="hover:bg-secondary/10 transition-colors">
                    <td className="px-4 py-3 font-mono text-[11px]">{task.id}</td>
                    <td className="px-4 py-3">{task.name || '—'}</td>
                    <td className="px-4 py-3 text-muted-foreground font-mono text-[11px]">{task.workflowId}</td>
                    <td className="px-4 py-3">
                      <span
                        className={cn(
                          'px-1.5 py-0.5 rounded text-[9px] font-bold border uppercase',
                          task.status === 'RUNNING'
                            ? 'bg-blue-500/10 text-blue-600 border-blue-500/20'
                            : task.status === 'WAITING_FOR_APPROVAL'
                            ? 'bg-amber-500/10 text-amber-600 border-amber-500/20'
                            : 'bg-orange-500/10 text-orange-600 border-orange-500/20'
                        )}
                      >
                        {task.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {new Date(task.createdAt).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}
