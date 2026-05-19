import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Plus, Play, Trash2, Edit2, Search, GitBranch, Clock, Loader2,
  ChevronDown, ChevronRight, CheckCircle2, XCircle, AlertTriangle, Circle,
} from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import { cn } from '@/lib/utils';
import api from '@/api/client';
import type { Task } from '@/types';

// ─── Task status helpers ──────────────────────────────────────────────────────
function TaskStatusIcon({ status }: { status: string }) {
  const s = (status || '').toUpperCase();
  if (s === 'COMPLETED') return <CheckCircle2 size={13} className="text-green-500 shrink-0" />;
  if (s === 'FAILED')    return <XCircle size={13} className="text-red-500 shrink-0" />;
  if (s === 'RUNNING')   return <Loader2 size={13} className="animate-spin text-blue-500 shrink-0" />;
  if (s === 'WAITING_FOR_APPROVAL') return <AlertTriangle size={13} className="text-amber-500 shrink-0" />;
  return <Circle size={13} className="text-muted-foreground shrink-0" />;
}

function taskStatusBadge(status: string) {
  const s = (status || '').toUpperCase();
  if (s === 'COMPLETED') return 'bg-green-500/10 text-green-700 border-green-500/20';
  if (s === 'FAILED')    return 'bg-red-500/10 text-red-700 border-red-500/20';
  if (s === 'RUNNING')   return 'bg-blue-500/10 text-blue-700 border-blue-500/20';
  if (s === 'WAITING_FOR_APPROVAL') return 'bg-amber-500/10 text-amber-700 border-amber-500/20';
  return 'bg-gray-500/10 text-gray-600 border-gray-500/20';
}

// ─── Workflow Results Drawer ──────────────────────────────────────────────────
function WorkflowResultsRow({ workflowId }: { workflowId: string }) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedTask, setExpandedTask] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    api.get(`/workflows/${workflowId}`)
      .then((res) => {
        const taskMap: Record<string, Task> = res.data?.tasks ?? {};
        setTasks(Object.values(taskMap));
        setError(null);
      })
      .catch((err) => {
        setError(err?.response?.data?.error ?? err.message ?? 'Failed to load tasks');
      })
      .finally(() => setLoading(false));
  }, [workflowId]);

  if (loading) {
    return (
      <tr><td colSpan={5} className="px-8 py-4 bg-muted/20">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 size={12} className="animate-spin" /> Loading task results…
        </div>
      </td></tr>
    );
  }

  if (error) {
    return (
      <tr><td colSpan={5} className="px-8 py-4 bg-red-500/5 border-y border-red-500/10">
        <span className="text-xs text-red-600">{error}</span>
      </td></tr>
    );
  }

  if (tasks.length === 0) {
    return (
      <tr><td colSpan={5} className="px-8 py-3 bg-muted/10 text-xs text-muted-foreground italic">
        No tasks found for this workflow.
      </td></tr>
    );
  }

  return (
    <>
      {tasks.map((task) => (
        <>
          <tr
            key={task.id}
            className="bg-muted/10 hover:bg-muted/20 cursor-pointer"
            onClick={() => setExpandedTask(expandedTask === task.id ? null : task.id)}
          >
            <td colSpan={5} className="px-8 py-2.5 border-t border-border/40">
              <div className="flex items-center gap-3">
                <TaskStatusIcon status={task.status} />
                <span className="text-xs font-medium text-foreground">{task.name || task.id}</span>
                <span className={cn('px-1.5 py-0.5 rounded text-[9px] font-bold border uppercase', taskStatusBadge(task.status))}>
                  {task.status}
                </span>
                {task.provider && (
                  <span className="text-[10px] text-muted-foreground ml-auto mr-2">
                    {task.provider} / {task.model}
                  </span>
                )}
                {(task.output || task.error) && (
                  expandedTask === task.id
                    ? <ChevronDown size={12} className="text-muted-foreground ml-auto" />
                    : <ChevronRight size={12} className="text-muted-foreground ml-auto" />
                )}
              </div>
            </td>
          </tr>

          {expandedTask === task.id && (
            <tr key={task.id + '-detail'} className="bg-muted/5">
              <td colSpan={5} className="px-10 py-3 border-t border-border/30">
                <div className="space-y-3 max-w-3xl">
                  {task.error && (
                    <div className="bg-red-500/5 border border-red-500/20 rounded p-3">
                      <p className="text-[10px] font-semibold text-red-600 mb-1 uppercase tracking-wider">Error</p>
                      <pre className="text-xs text-red-700 font-mono whitespace-pre-wrap break-all">{task.error}</pre>
                    </div>
                  )}
                  {task.output && Object.keys(task.output).length > 0 && (
                    <div className="bg-green-500/5 border border-green-500/20 rounded p-3">
                      <p className="text-[10px] font-semibold text-green-700 mb-1 uppercase tracking-wider">Output</p>
                      <pre className="text-xs text-foreground font-mono whitespace-pre-wrap break-all">
                        {JSON.stringify(task.output, null, 2)}
                      </pre>
                    </div>
                  )}
                  {task.input && Object.keys(task.input).length > 0 && (
                    <div className="bg-muted/30 border border-border rounded p-3">
                      <p className="text-[10px] font-semibold text-muted-foreground mb-1 uppercase tracking-wider">Input</p>
                      <pre className="text-xs text-foreground font-mono whitespace-pre-wrap break-all">
                        {JSON.stringify(task.input, null, 2)}
                      </pre>
                    </div>
                  )}
                  {task.startedAt && (
                    <p className="text-[10px] text-muted-foreground">
                      Started: {new Date(task.startedAt).toLocaleString()}
                      {task.finishedAt && ` · Finished: ${new Date(task.finishedAt).toLocaleString()}`}
                    </p>
                  )}
                </div>
              </td>
            </tr>
          )}
        </>
      ))}
    </>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────
export default function WorkflowsPage() {
  const { workflows, fetchWorkflows, deleteWorkflow, executeWorkflow } = useWorkflowStore();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [actionInProgress, setActionInProgress] = useState<string | null>(null);
  const [expandedResults, setExpandedResults] = useState<string | null>(null);

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  const getStatusBadge = useCallback((status?: string) => {
    switch ((status || '').toUpperCase()) {
      case 'COMPLETED': return 'bg-green-500/10 text-green-600 border-green-500/20';
      case 'RUNNING':   return 'bg-blue-500/10 text-blue-600 border-blue-500/20';
      case 'FAILED':    return 'bg-red-500/10 text-red-600 border-red-500/20';
      default:          return 'bg-gray-500/10 text-gray-600 border-gray-500/20';
    }
  }, []);

  const filteredWorkflows = workflows.data.filter((wf) => {
    const s = (search || '').toLowerCase();
    return (wf.name || '').toLowerCase().includes(s) || (wf.description || '').toLowerCase().includes(s);
  });

  const handleRun = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setActionInProgress(id + '-run');
    await executeWorkflow(id);
    // Refresh after a moment so status updates
    setTimeout(() => fetchWorkflows(), 1500);
    setActionInProgress(null);
  };

  const handleDelete = async (id: string, name: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm(`Delete workflow "${name}"? This cannot be undone.`)) return;
    if (expandedResults === id) setExpandedResults(null);
    setActionInProgress(id + '-delete');
    await deleteWorkflow(id);
    setActionInProgress(null);
  };

  const toggleResults = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setExpandedResults(expandedResults === id ? null : id);
  };

  if (workflows.error) {
    return (
      <div className="p-6 max-w-[1600px] mx-auto space-y-6">
        <div className="bg-red-500/10 border border-red-500/20 text-red-500 p-4 rounded-lg">
          <h2 className="font-bold mb-1">Failed to Load Workflows</h2>
          <p className="text-sm font-mono">{workflows.error}</p>
          <button onClick={fetchWorkflows} className="mt-3 text-xs font-medium text-red-600 hover:underline">
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
          <p className="text-sm text-muted-foreground">Manage, monitor, and inspect execution results.</p>
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
          <button
            onClick={() => fetchWorkflows()}
            disabled={workflows.isLoading}
            className="text-xs text-muted-foreground hover:text-foreground border border-border px-2.5 py-1.5 rounded-md hover:bg-secondary transition-colors disabled:opacity-50"
          >
            {workflows.isLoading ? <Loader2 size={12} className="animate-spin" /> : 'Refresh'}
          </button>
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
              <button onClick={() => navigate('/workflows/new')} className="text-xs font-medium text-primary hover:underline">
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
                  const taskCount = workflow.taskCount ?? (workflow.tasks ? Object.keys(workflow.tasks).length : 0);
                  const isExpanded = expandedResults === workflow.id;
                  const hasResults = ['COMPLETED', 'FAILED', 'RUNNING'].includes((workflow.status || '').toUpperCase());
                  return (
                    <>
                      <tr
                        key={workflow.id}
                        className={cn('hover:bg-secondary/20 transition-colors group', isExpanded && 'bg-secondary/10')}
                      >
                        <td className="px-4 py-3">
                          <div className="flex flex-col">
                            <button
                              onClick={(e) => hasResults && toggleResults(workflow.id, e)}
                              className={cn(
                                'font-semibold text-foreground text-left flex items-center gap-1.5',
                                hasResults && 'hover:text-primary cursor-pointer'
                              )}
                              title={hasResults ? 'Click to view task results' : undefined}
                            >
                              {hasResults && (
                                isExpanded
                                  ? <ChevronDown size={13} className="text-primary shrink-0" />
                                  : <ChevronRight size={13} className="text-muted-foreground shrink-0" />
                              )}
                              {workflow.name}
                            </button>
                            <span className="text-xs text-muted-foreground truncate max-w-[300px] ml-5">
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
                            {hasResults && (
                              <button
                                onClick={(e) => toggleResults(workflow.id, e)}
                                className={cn(
                                  'p-1.5 text-muted-foreground hover:text-primary hover:bg-secondary rounded transition-colors text-[10px] font-medium px-2',
                                  isExpanded && 'text-primary bg-primary/5'
                                )}
                                title="View task results"
                              >
                                {isExpanded ? 'Hide' : 'Results'}
                              </button>
                            )}
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

                      {isExpanded && (
                        <WorkflowResultsRow key={workflow.id + '-results'} workflowId={workflow.id} />
                      )}
                    </>
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
