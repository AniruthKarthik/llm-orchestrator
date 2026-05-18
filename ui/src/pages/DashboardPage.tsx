import { useEffect } from 'react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { GitBranch, Activity, Clock, Box, ShieldCheck, Cpu } from 'lucide-react';
import { Link } from 'react-router-dom';
import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';

const mockThroughputData = [
  { time: '00:00', tasks: 120, tokens: 45000 },
  { time: '04:00', tasks: 300, tokens: 120000 },
  { time: '08:00', tasks: 450, tokens: 180000 },
  { time: '12:00', tasks: 600, tokens: 250000 },
  { time: '16:00', tasks: 400, tokens: 160000 },
  { time: '20:00', tasks: 200, tokens: 80000 },
  { time: '24:00', tasks: 150, tokens: 60000 },
];

export default function DashboardPage() {
  const { workflows, fetchWorkflows } = useWorkflowStore();

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Overview</h1>
          <p className="text-sm text-muted-foreground">Real-time metrics across all orchestration nodes.</p>
        </div>
        <div className="flex items-center gap-2">
           <select className="h-8 text-xs border border-border bg-card rounded px-2 outline-none">
             <option>Last 24 Hours</option>
             <option>Last 7 Days</option>
             <option>Last 30 Days</option>
           </select>
        </div>
      </div>

      {/* Top Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
        {[
          { label: 'Active Workflows', value: workflows.length.toString(), sub: '+2 from yesterday', icon: <GitBranch size={16} className="text-blue-500" /> },
          { label: 'Tasks in Queue', value: '1,204', sub: 'Healthy (0 dead-letter)', icon: <Box size={16} className="text-orange-500" /> },
          { label: 'Total Tokens', value: '4.2M', sub: '~$42.00 est. cost', icon: <Cpu size={16} className="text-purple-500" /> },
          { label: 'Avg Latency', value: '840ms', sub: '-120ms from avg', icon: <Clock size={16} className="text-green-500" /> },
          { label: 'Provider Health', value: '100%', sub: '4/4 providers active', icon: <ShieldCheck size={16} className="text-emerald-500" /> },
        ].map((stat) => (
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
        {/* Main Chart */}
        <div className="lg:col-span-2 bg-card border border-border rounded-lg shadow-sm">
          <div className="p-4 border-b border-border flex items-center justify-between">
            <h3 className="font-semibold text-sm">Execution Throughput & Tokens</h3>
            <div className="flex items-center gap-4 text-xs">
              <div className="flex items-center gap-1.5"><div className="w-2 h-2 rounded-full bg-blue-500"></div>Tasks</div>
              <div className="flex items-center gap-1.5"><div className="w-2 h-2 rounded-full bg-purple-500"></div>Tokens</div>
            </div>
          </div>
          <div className="p-4 h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={mockThroughputData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="colorTasks" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e4e4e7" />
                <XAxis dataKey="time" axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#71717a' }} />
                <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 12, fill: '#71717a' }} />
                <Tooltip contentStyle={{ borderRadius: '8px', border: '1px solid #e4e4e7', fontSize: '12px' }} />
                <Area type="monotone" dataKey="tasks" stroke="#3b82f6" strokeWidth={2} fillOpacity={1} fill="url(#colorTasks)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Recent Workflows */}
        <div className="bg-card border border-border rounded-lg shadow-sm flex flex-col">
          <div className="p-4 border-b border-border flex items-center justify-between">
            <h3 className="font-semibold text-sm">Recent Executions</h3>
            <Link to="/executions" className="text-xs text-primary hover:underline font-medium">View All</Link>
          </div>
          <div className="flex-1 overflow-auto p-2 space-y-1">
            {workflows.slice(0, 6).map((wf) => (
              <div key={wf.id} className="p-2.5 rounded-md hover:bg-secondary transition-colors flex items-center justify-between group">
                <div className="flex flex-col gap-1">
                  <span className="font-semibold text-sm truncate max-w-[180px]">{wf.name}</span>
                  <span className="text-xs text-muted-foreground">{new Date(wf.createdAt).toLocaleTimeString()}</span>
                </div>
                <div className="flex items-center gap-3">
                   <div className="text-xs font-mono text-muted-foreground group-hover:text-foreground transition-colors">1.2s</div>
                   <div className="px-2 py-0.5 rounded text-[10px] font-bold bg-green-500/10 text-green-600 border border-green-500/20">
                     SUCCESS
                   </div>
                </div>
              </div>
            ))}
            {workflows.length === 0 && (
              <div className="p-8 text-center text-xs text-muted-foreground flex flex-col items-center gap-2">
                 <Activity size={24} className="opacity-20" />
                 No executions yet
              </div>
            )}
          </div>
        </div>
      </div>
      
      {/* Infrastructure Health */}
      <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden">
         <div className="p-4 border-b border-border">
            <h3 className="font-semibold text-sm">Infrastructure Nodes</h3>
         </div>
         <div className="grid grid-cols-1 md:grid-cols-4 divide-y md:divide-y-0 md:divide-x divide-border">
            {[
              { name: 'Core Engine', status: 'Healthy', ping: '12ms', ip: '10.0.1.1' },
              { name: 'Redis Queue', status: 'Healthy', ping: '4ms', ip: '10.0.1.5' },
              { name: 'Postgres Store', status: 'Healthy', ping: '8ms', ip: '10.0.1.12' },
              { name: 'Worker Group A', status: '8/8 Nodes', ping: 'N/A', ip: 'Scale Set' },
            ].map((node) => (
               <div key={node.name} className="p-4 flex flex-col gap-1">
                  <div className="flex items-center justify-between">
                     <span className="text-sm font-semibold">{node.name}</span>
                     <div className="w-2 h-2 rounded-full bg-green-500"></div>
                  </div>
                  <div className="text-xs text-muted-foreground flex items-center justify-between mt-2">
                     <span>{node.ip}</span>
                     <span className="font-mono">{node.ping}</span>
                  </div>
               </div>
            ))}
         </div>
      </div>
    </div>
  );
}
