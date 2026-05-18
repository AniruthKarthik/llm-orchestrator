import { useEffect, useState } from 'react';
import { Shield, Cpu, Search, Users, Loader2 } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';

export default function AgentsPage() {
  const { agents, fetchAgents } = useWorkflowStore();
  const [search, setSearch] = useState('');

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  const filtered = agents.data.filter(
    (a) =>
      a.Name?.toLowerCase().includes(search.toLowerCase()) ||
      a.Role?.toLowerCase().includes(search.toLowerCase()) ||
      a.Provider?.toLowerCase().includes(search.toLowerCase()) ||
      a.ID?.toLowerCase().includes(search.toLowerCase())
  );

  if (agents.isLoading) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto flex justify-center items-center h-[50vh]">
        <Loader2 className="animate-spin h-8 w-8 text-primary" />
      </div>
    );
  }

  if (agents.error) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto">
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-lg">
          <h2 className="font-bold mb-1">Failed to Load Agents</h2>
          <p className="text-sm font-mono">{agents.error}</p>
          <button onClick={fetchAgents} className="mt-3 text-xs font-medium text-red-600 hover:underline">
            Retry →
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Agent Registry</h1>
          <p className="text-sm text-muted-foreground">
            Manage specialized agents and their runtime capabilities.
          </p>
        </div>
        <div className="text-xs text-muted-foreground bg-secondary px-3 py-1 rounded border border-border">
          {agents.data.length} registered
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden flex flex-col">
        <div className="p-3 border-b border-border flex items-center gap-3 bg-muted/20">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-2.5 top-2 text-muted-foreground" size={14} />
            <input
              type="text"
              placeholder="Filter by name, role, provider, or ID..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-8 w-full bg-background border border-border rounded-md pl-8 pr-3 text-xs focus:outline-none focus:ring-1 focus:ring-primary transition-colors"
            />
          </div>
        </div>

        {filtered.length === 0 ? (
          <div className="py-20 text-center">
            <Users size={48} className="mx-auto text-muted-foreground opacity-20 mb-4" />
            <p className="text-muted-foreground text-sm">
              {search ? 'No agents match your search.' : 'No registered agents found in the registry.'}
            </p>
            {!search && (
              <p className="text-xs text-muted-foreground mt-2">
                Agents are registered when you save and execute a workflow with a provider assigned.
              </p>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-muted/50 border-b border-border text-[11px] text-muted-foreground uppercase tracking-wider font-bold">
                  <th className="px-4 py-3">Agent Name / ID</th>
                  <th className="px-4 py-3">Role</th>
                  <th className="px-4 py-3">Intelligence Source</th>
                  <th className="px-4 py-3">Tools</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border text-sm font-medium">
                {filtered.map((agent) => (
                  <tr key={agent.ID} className="hover:bg-secondary/20 transition-colors">
                    <td className="px-4 py-4">
                      <div className="flex flex-col">
                        <span className="text-foreground">{agent.Name || '—'}</span>
                        <span className="text-[10px] text-muted-foreground font-mono">{agent.ID}</span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <Shield size={14} className="text-blue-500" />
                        <span className="text-xs">{agent.Role || '—'}</span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <Cpu size={14} className="text-purple-500" />
                        <span className="text-xs">
                          {agent.Provider ? `${agent.Provider}${agent.Model ? ` (${agent.Model})` : ''}` : '—'}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex flex-wrap gap-1">
                        {(agent.Tools || []).length > 0 ? (
                          agent.Tools.map((cap) => (
                            <span
                              key={cap}
                              className="px-1.5 py-0.5 bg-secondary text-[10px] rounded border border-border text-muted-foreground"
                            >
                              {cap}
                            </span>
                          ))
                        ) : (
                          <span className="text-[10px] text-muted-foreground italic">Generic (no tools)</span>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
