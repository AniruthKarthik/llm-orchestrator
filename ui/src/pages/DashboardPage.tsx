import { useEffect } from 'react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { useWebSocket } from '@/hooks/useWebSocket';
import { GitBranch, Activity, Box, ShieldCheck, Cpu, Loader2 } from 'lucide-react';
import { Link } from 'react-router-dom';
import { cn } from '@/lib/utils';

const statusColors: Record<string, string> = {
  COMPLETED: 'text-green-600 bg-green-500/10 border-green-500/20',
  RUNNING: 'text-blue-600 bg-blue-500/10 border-blue-500/20',
  FAILED: 'text-red-600 bg-red-500/10 border-red-500/20',
  PENDING: 'text-gray-600 bg-gray-500/10 border-gray-500/20',
};

export default function DashboardPage() {
  const { workflows, agents, metrics, metricsLoading, fetchWorkflows, fetchAgents, fetchMetrics } = useWorkflowStore();
  const { events } = useWebSocket();

  useEffect(() => {
    fetchWorkflows();
    fetchAgents();
    fetchMetrics();
  }, [fetchWorkflows, fetchAgents, fetchMetrics]);

  const statsCards = [
    {
      label: 'Active Workflows',
      value: metrics?.activeWorkflows ?? workflows.data.filter((w) => w.status === 'RUNNING').length,
      sub: 'Currently executing',
      icon: <GitBranch size={16} className="text-blue-500" />,
    },
    {
      label: 'Tasks in Queue',
      value: metrics?.tasksInQueue ?? 0,
      sub: 'Pending/running tasks',
      icon: <Box size={16} className="text-orange-500" />,
    },
    {
      label: 'Registered Agents',
      value: agents.data.length,
      sub: 'Active agent definitions',
      icon: <Cpu size={16} className="text-purple-500" />,
    },
    {
      label: 'Live Events',
      value: events.length,
      sub: 'This session',
      icon: <Activity size={16} className="text-green-500" />,
    },
    {
      label: 'Providers Online',
      value: `${metrics?.providersOnline ?? 0}`,
      sub: `${metrics?.providersOnline ?? 0} configured`,
      icon: <ShieldCheck size={16} className="text-emerald-500" />,
    },
  ];

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Overview</h1>
          <p className="text-sm text-muted-foreground">Real-time metrics across all orchestration nodes.</p>
        </div>
        {metricsLoading && (
          <Loader2 size={16} className="animate-spin text-muted-foreground" />
        )}
      </div>

      {/* Top Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
        {statsCards.map((stat) => (
          <div key={stat.label} className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">{stat.label}</span>
              {stat.icon}
            </div>
            <div className="text-2xl font-bold mb-1">{stat.value}</div>
            <div className="text-xs text-muted-foreground">{stat.sub}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Summary stats */}
        <div className="lg:col-span-2 bg-card border border-border rounded-lg shadow-sm">
          <div className="p-4 border-b border-border flex items-center justify-between">
            <h3 className="font-semibold text-sm">Workflow Summary</h3>
          </div>
          {metrics ? (
            <div className="grid grid-cols-2 sm:grid-cols-4 divide-x divide-y sm:divide-y-0 divide-border">
              {[
                { label: 'Total', value: metrics.totalWorkflows, color: 'text-foreground' },
                { label: 'Active', value: metrics.activeWorkflows, color: 'text-blue-600' },
                { label: 'Completed', value: metrics.completedWorkflows, color: 'text-green-600' },
                { label: 'Failed', value: metrics.failedWorkflows, color: 'text-red-600' },
              ].map((item) => (
                <div key={item.label} className="p-6 text-center">
                  <div className={cn('text-3xl font-bold', item.color)}>{item.value}</div>
                  <div className="text-xs text-muted-foreground mt-1 uppercase tracking-wider">{item.label}</div>
                </div>
              ))}
            </div>
          ) : (
            <div className="p-8 h-[180px] flex flex-col items-center justify-center text-center text-muted-foreground bg-muted/10">
              {metricsLoading ? (
                <Loader2 size={24} className="animate-spin opacity-30 mb-2" />
              ) : (
                <>
                  <Activity className="opacity-20 mb-3" size={32} />
                  <p className="font-medium text-sm">No metrics available</p>
                  <p className="text-xs mt-1">Could not connect to the backend metrics endpoint.</p>
                </>
              )}
            </div>
          )}
        </div>

        {/* Recent Workflows */}
        <div className="bg-card border border-border rounded-lg shadow-sm flex flex-col">
          <div className="p-4 border-b border-border flex items-center justify-between">
            <h3 className="font-semibold text-sm">Recent Executions</h3>
            <Link to="/executions" className="text-xs text-primary hover:underline font-medium">View All</Link>
          </div>
          <div className="flex-1 overflow-auto p-2 space-y-1">
            {workflows.isLoading ? (
              <div className="p-8 flex justify-center">
                <Loader2 size={20} className="animate-spin text-muted-foreground" />
              </div>
            ) : workflows.data.length === 0 ? (
              <div className="p-8 text-center text-xs text-muted-foreground flex flex-col items-center gap-2">
                <Activity size={24} className="opacity-20" />
                No executions yet
              </div>
            ) : (
              workflows.data.slice(0, 10).map((wf) => (
                <Link
                  key={wf.id}
                  to={`/workflows/${wf.id}`}
                  className="p-2.5 rounded-md hover:bg-secondary transition-colors flex items-center justify-between group block"
                >
                  <div className="flex flex-col gap-1">
                    <span className="font-semibold text-sm truncate max-w-[180px]">{wf.name}</span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(wf.createdAt).toLocaleTimeString()}
                    </span>
                  </div>
                  <span className={cn(
                    'px-2 py-0.5 rounded text-[10px] font-bold border uppercase',
                    statusColors[wf.status?.toUpperCase()] || statusColors['PENDING']
                  )}>
                    {wf.status || 'UNKNOWN'}
                  </span>
                </Link>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
