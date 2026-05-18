import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Play, Trash2, Edit2, Search, GitBranch, Clock, Loader2 } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { cn } from '@/lib/utils';

export default function WorkflowsPage() {
  const { workflows, fetchWorkflows, deleteWorkflow, executeWorkflow } = useWorkflowStore();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [actionInProgress, setActionInProgress] = useState<string | null>(null);

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  const getStatusBadge = (status: string) => {
    switch (status.toUpperCase()) {
      case 'COMPLETED': return 'bg-green-500/10 text-green-600 border-green-500/20';
      case 'RUNNING': return 'bg-blue-500/10 text-blue-600 border-blue-500/20';
      case 'FAILED': return 'bg-red-500/10 text-red-600 border-red-500/20';
      default: return 'bg-gray-500/10 text-gray-600 border-gray-500/20';
    }
  };

  const filteredWorkflows = workflows.data.filter((wf) =>
    wf.name.toLowerCase().includes(search.toLowerCase()) ||
    (wf.description && wf.description.toLowerCase().includes(search.toLowerCase()))
  );

  const handleRun = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setActionInProgress(id + '-run');
    await executeWorkflow(id);
    setActionInProgress(null);
  };

  const handleDelete = async (id: string, name: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm(`Delete workflow "${name}"? This cannot be undone.`)) return;
    setActionInProgress(id + '-delete');
    await deleteWorkflow(id);
    setActionInProgress(null);
  };

  if (workflows.error) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto space-y-6">
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-lg">
          <h2 className="font-bold mb-1">Failed to Load Workflows</h2>
          <p className="text-sm font-mono">{workflows.error}</p>
          <button
            onClick={fetchWorkflows}
            className="mt-3 text-xs font-medium text-red-600 hover:underline"
          >
            Retry →
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-[1600px] mx-auto space-y-4">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-xl font-bold text-foreground">Workflows</h1>
          <p className="text-sm text-muted-foreground">Manage, monitor, and configure execution pipelines.</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => navigate('/workflows/new')}
            className="flex items-center gap-2 bg-primary text-primary-foreground px-3 py-1.5 rounded-md font-medium text-xs hover:bg-primary/90 transition-colors"
          >
            <Plus size={14} />
            New Workflow
          </button>
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden flex flex-col">
        <div className="p-3 border-b border-border flex items-center gap-3">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-2.5 top-2 text-muted-foreground" size={14} />
            <input
              type="text"
              placeholder="Search workflows by name or description..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="h-8 w-full bg-background border border-border rounded-md pl-8 pr-3 text-xs focus:outline-none focus:ring-1 focus:ring-primary transition-colors"
            />
          </div>
        </div>

        {workflows.isLoading ? (
          <div className="flex justify-center items-center py-20 text-muted-foreground">
            <Loader2 size={20} className="animate-spin mr-3" />
            <span className="text-sm font-medium">Loading workflows...</span>
          </div>
        ) : filteredWorkflows.length === 0 ? (
          <div className="text-center py-20 bg-background/50">
            <GitBranch size={32} className="mx-auto text-muted-foreground mb-3 opacity-20" />
            <h3 className="text-sm font-semibold text-foreground mb-1">
              {search ? 'No workflows match your search' : 'No workflows yet'}
            </h3>
            <p className="text-xs text-muted-foreground mb-4">
              {search ? 'Try a different search term.' : "You haven't created any workflows yet."}
            </p>
            {!search && (
              <button
                onClick={() => navigate('/workflows/new')}
                className="text-xs font-medium text-primary hover:underline"
              >
                Create your first workflow →
              </button>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-secondary/50 border-b border-border text-xs text-muted-foreground uppercase tracking-wider">
                  <th className="px-4 py-2 font-semibold">Workflow Name</th>
                  <th className="px-4 py-2 font-semibold">Status</th>
                  <th className="px-4 py-2 font-semibold">Created</th>
                  <th className="px-4 py-2 font-semibold">Tasks</th>
                  <th className="px-4 py-2 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border text-sm">
                {filteredWorkflows.map((workflow) => {
                  const taskCount = workflow.tasks ? Object.keys(workflow.tasks).length : 0;
                  return (
                    <tr key={workflow.id} className="hover:bg-secondary/20 transition-colors group">
                      <td className="px-4 py-3">
                        <div className="flex flex-col">
                          <span className="font-semibold text-foreground">{workflow.name}</span>
                          <span className="text-xs text-muted-foreground truncate max-w-[300px]">
                            {workflow.description || workflow.id}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={cn('px-2 py-0.5 rounded text-[10px] font-bold border uppercase', getStatusBadge(workflow.status || 'UNKNOWN'))}>
                          {workflow.status || 'UNKNOWN'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground text-xs">
                        <div className="flex items-center gap-1.5">
                          <Clock size={12} />
                          {new Date(workflow.createdAt).toLocaleString()}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground text-xs">
                        <div className="flex items-center gap-1.5">
                          <GitBranch size={12} />
                          {taskCount} {taskCount === 1 ? 'node' : 'nodes'}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button
                            onClick={() => navigate(`/workflows/${workflow.id}`)}
                            className="p-1.5 text-muted-foreground hover:text-primary hover:bg-secondary rounded transition-colors"
                            title="Edit Workflow"
                          >
                            <Edit2 size={14} />
                          </button>
                          <button
                            onClick={(e) => handleRun(workflow.id, e)}
                            disabled={actionInProgress === workflow.id + '-run' || workflow.status === 'RUNNING'}
                            className="p-1.5 text-muted-foreground hover:text-green-600 hover:bg-green-50 rounded transition-colors disabled:opacity-50"
                            title="Run Workflow"
                          >
                            {actionInProgress === workflow.id + '-run'
                              ? <Loader2 size={14} className="animate-spin" />
                              : <Play size={14} />}
                          </button>
                          <button
                            onClick={(e) => handleDelete(workflow.id, workflow.name, e)}
                            disabled={actionInProgress === workflow.id + '-delete' || workflow.status === 'RUNNING'}
                            className="p-1.5 text-muted-foreground hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50"
                            title="Delete Workflow"
                          >
                            {actionInProgress === workflow.id + '-delete'
                              ? <Loader2 size={14} className="animate-spin" />
                              : <Trash2 size={14} />}
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
