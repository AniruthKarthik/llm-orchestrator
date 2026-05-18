import { useState, useCallback, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ReactFlow,
  Background,
  Controls,
  Panel,
  useNodesState,
  useEdgesState,
  addEdge,
  MarkerType,
  BackgroundVariant,
  type Connection,
  type Edge,
  type Node,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import TaskNode from '@/components/builder/TaskNode';
import { Save, Play, Plus, ArrowLeft, Settings2, Loader2, AlertCircle } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import api from '@/api/client';
import { cn } from '@/lib/utils';

const nodeTypes = {
  task: TaskNode,
};

export default function WorkflowBuilderPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { providers, fetchProviders, executeWorkflow } = useWorkflowStore();

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isExecuting, setIsExecuting] = useState(false);
  const [workflowName, setWorkflowName] = useState('Untitled Workflow');
  const [workflowDesc, setWorkflowDesc] = useState('');
  const [activeTab, setActiveTab] = useState<'config' | 'prompt' | 'advanced'>('config');
  const [saveError, setSaveError] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [savedWorkflowId, setSavedWorkflowId] = useState<string | null>(id !== 'new' ? id ?? null : null);

  useEffect(() => {
    fetchProviders();
    if (id && id !== 'new') {
      loadWorkflow(id);
    }
  }, [id, fetchProviders]);

  const loadWorkflow = async (wfId: string) => {
    setLoadError(null);
    try {
      const response = await api.get(`/workflows/${wfId}`);
      const wf = response.data;
      setWorkflowName(wf.name);
      setWorkflowDesc(wf.description);

      const taskMap = wf.tasks || {};
      const taskArray = Object.values(taskMap) as Array<Record<string, unknown>>;

      // Simple auto-layout: horizontal flow
      const newNodes: Node[] = taskArray.map((task, index) => ({
        id: task.id as string,
        type: 'task',
        position: { x: (index % 4) * 300 + 50, y: Math.floor(index / 4) * 200 + 50 },
        data: {
          label: task.name,
          description: task.description,
          status: task.status || 'PENDING',
          config: (task.input as Record<string, unknown>) || {},
          provider: task.provider || '',
          model: task.model || '',
          systemPrompt: ((task.input as Record<string, unknown>)?.system_prompt as string) || '',
          maxRetries: 3,
          timeoutMs: 30000,
        },
      }));

      const newEdges: Edge[] = [];
      taskArray.forEach((task) => {
        const deps = (task.dependencies as string[]) || [];
        deps.forEach((depId) => {
          newEdges.push({
            id: `e-${depId}-${task.id}`,
            source: depId,
            target: task.id as string,
            type: 'smoothstep',
            markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--color-muted-foreground)' },
            style: { stroke: 'var(--color-border)', strokeWidth: 2 },
          });
        });
      });

      setNodes(newNodes);
      setEdges(newEdges);
    } catch (error: unknown) {
      const msg =
        error instanceof Error ? error.message : 'Failed to load workflow';
      setLoadError(msg);
      console.error('Failed to load workflow:', error);
    }
  };

  const onConnect = useCallback((params: Connection) => {
    setEdges((eds) =>
      addEdge(
        {
          ...params,
          type: 'smoothstep',
          markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--color-muted-foreground)' },
          style: { stroke: 'var(--color-border)', strokeWidth: 2 },
        },
        eds
      )
    );
  }, [setEdges]);

  const addNode = () => {
    const newNode: Node = {
      id: `task-${Date.now()}`,
      type: 'task',
      position: { x: 250, y: nodes.length * 150 + 100 },
      data: {
        label: 'New Task',
        status: 'PENDING',
        config: {},
        provider: '',
        model: '',
        systemPrompt: '',
        maxRetries: 3,
        timeoutMs: 30000,
      },
    };
    setNodes((nds) => nds.concat(newNode));
  };

  const onNodeClick = (_: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
    setActiveTab('config');
  };

  const updateNodeData = (nodeId: string, newData: Record<string, unknown>) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === nodeId) {
          return { ...node, data: { ...node.data, ...newData } };
        }
        return node;
      })
    );
    setSelectedNode((prev) => {
      if (!prev || prev.id !== nodeId) return prev;
      return { ...prev, data: { ...prev.data, ...newData } };
    });
  };

  const buildPayload = (overrideId?: string) => {
    const wfId = overrideId || savedWorkflowId || `wf-${Date.now()}`;
    return {
      id: wfId,
      name: workflowName.trim() || 'Untitled Workflow',
      description: workflowDesc,
      tasks: nodes.map((node) => {
        const deps = edges.filter((e) => e.target === node.id).map((e) => e.source);
        return {
          id: node.id,
          name: (node.data.label as string) || 'Unnamed Task',
          description: (node.data.description as string) || '',
          input: (node.data.config as Record<string, unknown>) || {},
          dependencies: deps,
          provider: (node.data.provider as string) || '',
          model: (node.data.model as string) || '',
          systemPrompt: (node.data.systemPrompt as string) || '',
          maxRetries: (node.data.maxRetries as number) ?? 3,
          timeoutMs: (node.data.timeoutMs as number) ?? 30000,
        };
      }),
    };
  };

  const saveWorkflow = async () => {
    if (workflowName.trim() === '') {
      setSaveError('Workflow name is required.');
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    try {
      const isNew = !savedWorkflowId || id === 'new';
      const payload = buildPayload();

      if (isNew) {
        await api.post('/workflows', payload);
        setSavedWorkflowId(payload.id);
        navigate(`/workflows/${payload.id}`, { replace: true });
      } else {
        await api.put(`/workflows/${savedWorkflowId}`, payload);
      }
    } catch (error: unknown) {
      let msg = 'Failed to save workflow.';
      if (error instanceof Error) msg = error.message;
      const axiosErr = error as Record<string, unknown>;
      if (axiosErr.response && typeof axiosErr.response === 'object') {
        const resp = axiosErr.response as Record<string, unknown>;
        if (resp.data && typeof resp.data === 'object') {
          const data = resp.data as Record<string, string>;
          if (data.error) msg = data.error;
        }
      }
      setSaveError(msg);
    } finally {
      setIsSaving(false);
    }
  };

  const handleExecute = async () => {
    if (!savedWorkflowId || id === 'new') {
      setSaveError('Please save the workflow before executing.');
      return;
    }
    setIsExecuting(true);
    setSaveError(null);
    const ok = await executeWorkflow(savedWorkflowId);
    if (!ok) {
      setSaveError('Failed to start workflow execution. Check provider configuration.');
    }
    setIsExecuting(false);
  };

  return (
    <div style={{ height: 'calc(100vh - 3rem)' }} className="w-full flex flex-col">
      {/* Top Toolbar */}
      <div className="h-12 border-b border-border bg-card px-4 flex items-center justify-between z-10 shrink-0 shadow-sm">
        <div className="flex items-center gap-2">
          <button
            onClick={() => navigate('/workflows')}
            className="p-1.5 hover:bg-secondary rounded text-muted-foreground transition-colors mr-2"
          >
            <ArrowLeft size={16} />
          </button>
          <div className="flex items-center gap-3">
            <input
              value={workflowName}
              onChange={(e) => setWorkflowName(e.target.value)}
              className="bg-transparent font-semibold text-sm focus:outline-none focus:ring-1 focus:ring-primary rounded px-1.5 py-0.5 w-[200px]"
              placeholder="Workflow Name"
            />
            <span className="text-xs text-muted-foreground bg-secondary px-1.5 py-0.5 rounded font-mono">
              {savedWorkflowId && id !== 'new' ? savedWorkflowId : 'Draft'}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {saveError && (
            <div className="flex items-center gap-1.5 text-xs text-red-600 bg-red-50 border border-red-200 rounded px-2 py-1 max-w-[260px]">
              <AlertCircle size={12} className="shrink-0" />
              <span className="truncate">{saveError}</span>
            </div>
          )}
          <button
            onClick={saveWorkflow}
            disabled={isSaving}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-border text-xs font-medium hover:bg-secondary transition-colors disabled:opacity-50"
          >
            {isSaving ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
            {isSaving ? 'Saving...' : 'Save Workflow'}
          </button>
          <div className="w-px h-4 bg-border mx-1" />
          <button
            onClick={handleExecute}
            disabled={isExecuting || !savedWorkflowId || id === 'new'}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-primary text-primary-foreground text-xs font-medium hover:opacity-90 transition-opacity disabled:opacity-50"
            title={(!savedWorkflowId || id === 'new') ? 'Save the workflow first' : 'Execute workflow'}
          >
            {isExecuting ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
            {isExecuting ? 'Executing...' : 'Execute'}
          </button>
        </div>
      </div>

      {loadError && (
        <div className="bg-red-500/10 border-b border-red-500/20 px-4 py-2 flex items-center gap-2 text-red-600 text-xs">
          <AlertCircle size={14} />
          <span>{loadError}</span>
          <button onClick={() => id && loadWorkflow(id)} className="ml-auto font-medium hover:underline">
            Retry
          </button>
        </div>
      )}

      <div className="flex-1 flex overflow-hidden">
        {/* Canvas Area */}
        <div className="flex-1 relative bg-background">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onPaneClick={() => setSelectedNode(null)}
            nodeTypes={nodeTypes as never}
            fitView
            className="[&_.react-flow__controls-button]:bg-card [&_.react-flow__controls-button]:border-border [&_.react-flow__controls-button]:fill-foreground"
          >
            <Background color="var(--color-border)" gap={24} variant={BackgroundVariant.Dots} />
            <Controls className="!mb-6 !ml-6" />
            <Panel position="top-left" className="!mt-4 !ml-4">
              <button
                onClick={addNode}
                className="flex items-center gap-1.5 bg-card border border-border shadow-sm px-3 py-1.5 rounded-md text-sm font-medium hover:bg-secondary transition-colors"
              >
                <Plus size={16} />
                Add Task Node
              </button>
            </Panel>
          </ReactFlow>
        </div>

        {/* Task Properties Inspector */}
        {selectedNode && (
          <div className="w-[380px] border-l border-border bg-card flex flex-col shadow-xl z-20 shrink-0">
            <div className="h-12 border-b border-border flex items-center justify-between px-4 shrink-0 bg-muted/30">
              <h3 className="font-semibold text-sm flex items-center gap-2">
                <Settings2 size={16} className="text-muted-foreground" />
                Task Configuration
              </h3>
              <button
                onClick={() => setSelectedNode(null)}
                className="text-muted-foreground hover:text-foreground p-1 rounded hover:bg-secondary"
              >
                &times;
              </button>
            </div>

            <div className="flex border-b border-border px-2 pt-2 gap-1 bg-muted/10 shrink-0">
              {(['config', 'prompt', 'advanced'] as const).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  className={cn(
                    'px-3 py-1.5 text-xs font-medium border-b-2 transition-colors capitalize',
                    activeTab === tab
                      ? 'border-primary text-foreground'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  )}
                >
                  {tab === 'config' ? 'General' : tab === 'prompt' ? 'Prompt' : 'Advanced'}
                </button>
              ))}
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-5 text-sm">
              {activeTab === 'config' && (
                <>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-foreground">Task Name</label>
                    <input
                      value={(selectedNode.data.label as string) || ''}
                      onChange={(e) => updateNodeData(selectedNode.id, { label: e.target.value })}
                      className="w-full bg-background border border-border rounded-md px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>

                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-foreground">Description</label>
                    <textarea
                      value={(selectedNode.data.description as string) || ''}
                      onChange={(e) => updateNodeData(selectedNode.id, { description: e.target.value })}
                      className="w-full bg-background border border-border rounded-md px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary h-20 resize-none"
                    />
                  </div>

                  <div className="space-y-1.5 pt-2 border-t border-border">
                    <label className="text-xs font-semibold text-foreground">Provider</label>
                    <select
                      value={(selectedNode.data.provider as string) || ''}
                      onChange={(e) => updateNodeData(selectedNode.id, { provider: e.target.value, model: '' })}
                      className="w-full bg-background border border-border rounded-md px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">Auto-Route (Recommended)</option>
                      {providers.data.map((p) => (
                        <option key={p.name} value={p.name}>{p.name}</option>
                      ))}
                    </select>
                  </div>

                  {!!selectedNode.data.provider && (
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold text-foreground">Model</label>
                      <select
                        value={(selectedNode.data.model as string) || ''}
                        onChange={(e) => updateNodeData(selectedNode.id, { model: e.target.value })}
                        className="w-full bg-background border border-border rounded-md px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                      >
                        <option value="">Default for Provider</option>
                        {providers.data
                          .find((p) => p.name === selectedNode.data.provider)
                          ?.models.map((m) => (
                            <option key={m} value={m}>{m}</option>
                          ))}
                      </select>
                    </div>
                  )}
                </>
              )}

              {activeTab === 'prompt' && (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-foreground">System Prompt</label>
                    <p className="text-[10px] text-muted-foreground">
                      Instructions sent as the system message to the LLM before task execution.
                    </p>
                    <textarea
                      value={(selectedNode.data.systemPrompt as string) || ''}
                      onChange={(e) => updateNodeData(selectedNode.id, { systemPrompt: e.target.value })}
                      className="w-full bg-secondary/30 font-mono text-xs border border-border rounded-md px-2.5 py-2 focus:outline-none focus:ring-1 focus:ring-primary h-48 resize-y"
                      placeholder="You are an expert orchestration agent tasked with..."
                    />
                  </div>
                </div>
              )}

              {activeTab === 'advanced' && (
                <div className="space-y-4">
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-foreground">Retry Policy</label>
                    <div className="grid grid-cols-2 gap-2">
                      <div>
                        <label className="text-[10px] text-muted-foreground block mb-1">Max Retries</label>
                        <input
                          type="number"
                          min={0}
                          max={10}
                          value={(selectedNode.data.maxRetries as number) ?? 3}
                          onChange={(e) => updateNodeData(selectedNode.id, { maxRetries: parseInt(e.target.value, 10) || 0 })}
                          className="w-full bg-background border border-border rounded-md px-2.5 py-1 text-xs"
                        />
                      </div>
                      <div>
                        <label className="text-[10px] text-muted-foreground block mb-1">Timeout (ms)</label>
                        <input
                          type="number"
                          min={1000}
                          step={1000}
                          value={(selectedNode.data.timeoutMs as number) ?? 30000}
                          onChange={(e) => updateNodeData(selectedNode.id, { timeoutMs: parseInt(e.target.value, 10) || 30000 })}
                          className="w-full bg-background border border-border rounded-md px-2.5 py-1 text-xs"
                        />
                      </div>
                    </div>
                  </div>

                  <div className="pt-8">
                    <button
                      onClick={() => {
                        setNodes((nds) => nds.filter((n) => n.id !== selectedNode.id));
                        setEdges((eds) => eds.filter((e) => e.source !== selectedNode.id && e.target !== selectedNode.id));
                        setSelectedNode(null);
                      }}
                      className="w-full flex items-center justify-center gap-1.5 text-red-600 border border-red-200 bg-red-50 hover:bg-red-100 py-1.5 rounded-md text-xs font-medium transition-colors"
                    >
                      Remove Task Node
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
