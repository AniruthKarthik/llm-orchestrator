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
import { Save, Play, Plus, ArrowLeft, MoreVertical, LayoutGrid, Settings2, Key } from 'lucide-react';
import { useWorkflowStore } from '@/store/useWorkflowStore';
import api from '@/api/client';
import { cn } from '@/lib/utils';

const nodeTypes = {
  task: TaskNode,
};

export default function WorkflowBuilderPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { providers, fetchProviders } = useWorkflowStore();
  
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [workflowName, setWorkflowName] = useState('Untitled Workflow');
  const [workflowDesc, setWorkflowDesc] = useState('');
  const [activeTab, setActiveTab] = useState<'config' | 'prompt' | 'advanced'>('config');

  useEffect(() => {
    fetchProviders();
    if (id && id !== 'new') {
      loadWorkflow(id);
    }
  }, [id, fetchProviders]);

  const loadWorkflow = async (wfId: string) => {
    try {
      const response = await api.get(`/workflows/${wfId}`);
      const wf = response.data;
      setWorkflowName(wf.name);
      setWorkflowDesc(wf.description);
      
      // Transform tasks to nodes
      const newNodes: Node[] = Object.values(wf.tasks || {}).map((task: any, index: number) => ({
        id: task.id,
        type: 'task',
        position: { x: 250, y: index * 150 + 50 },
        data: { 
          label: task.name, 
          description: task.description, 
          status: task.status || 'PENDING',
          config: task.input || {},
          provider: task.provider || '',
          model: task.model || '',
        },
      }));
      
      // Transform dependencies to edges
      const newEdges: Edge[] = [];
      Object.values(wf.tasks || {}).forEach((task: any) => {
        (task.dependencies || []).forEach((depId: string) => {
          newEdges.push({
            id: `e-${depId}-${task.id}`,
            source: depId,
            target: task.id,
            type: 'smoothstep',
            markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--color-muted-foreground)' },
            style: { stroke: 'var(--color-border)', strokeWidth: 2 },
          });
        });
      });
      
      setNodes(newNodes);
      setEdges(newEdges);
    } catch (error) {
      console.error('Failed to load workflow', error);
    }
  };

  const onConnect = useCallback((params: Connection) => {
    setEdges((eds) => addEdge({
      ...params,
      type: 'smoothstep',
      markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--color-muted-foreground)' },
      style: { stroke: 'var(--color-border)', strokeWidth: 2 },
    }, eds));
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
      },
    };
    setNodes((nds) => nds.concat(newNode));
  };

  const onNodeClick = (_: any, node: Node) => {
    setSelectedNode(node);
  };

  const updateNodeData = (nodeId: string, newData: any) => {
    setNodes((nds) => nds.map((node) => {
      if (node.id === nodeId) {
        return { ...node, data: { ...node.data, ...newData } };
      }
      return node;
    }));
    if (selectedNode?.id === nodeId) {
      setSelectedNode((prev: any) => ({ ...prev, data: { ...prev.data, ...newData } }));
    }
  };

  const saveWorkflow = async () => {
    setIsSaving(true);
    try {
      const payload = {
        id: id === 'new' ? `wf-${Date.now()}` : id,
        name: workflowName,
        description: workflowDesc,
        tasks: nodes.map(node => {
          const deps = edges
            .filter(edge => edge.target === node.id)
            .map(edge => edge.source);
          
          return {
            id: node.id,
            name: node.data.label,
            description: (node.data.description as string) || '',
            input: (node.data.config as any) || {},
            dependencies: deps,
            provider: (node.data.provider as string) || '',
            model: (node.data.model as string) || ''
          };
        })
      };
      
      if (id === 'new') {
        await api.post('/workflows', payload);
        navigate(`/workflows/${payload.id}`, { replace: true });
      } else {
        await api.put(`/workflows/${id}`, payload);
      }
    } catch (error) {
      console.error('Failed to save', error);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="h-full flex flex-col -m-6 sm:-m-8">
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
               {id === 'new' ? 'Draft' : id}
             </span>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          <button 
            onClick={saveWorkflow}
            disabled={isSaving}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-border text-xs font-medium hover:bg-secondary transition-colors disabled:opacity-50"
          >
            <Save size={14} />
            {isSaving ? 'Saving...' : 'Save Workflow'}
          </button>
          <div className="w-px h-4 bg-border mx-1" />
          <button 
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-primary text-primary-foreground text-xs font-medium hover:opacity-90 transition-opacity"
          >
            <Play size={14} />
            Execute
          </button>
          <button className="p-1.5 text-muted-foreground hover:bg-secondary rounded ml-1">
             <MoreVertical size={16} />
          </button>
        </div>
      </div>

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
            nodeTypes={nodeTypes as any}
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
              <button onClick={() => setSelectedNode(null)} className="text-muted-foreground hover:text-foreground p-1 rounded hover:bg-secondary">
                &times;
              </button>
            </div>

            <div className="flex border-b border-border px-2 pt-2 gap-1 bg-muted/10 shrink-0">
              <button 
                onClick={() => setActiveTab('config')}
                className={cn("px-3 py-1.5 text-xs font-medium border-b-2 transition-colors", activeTab === 'config' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground')}
              >
                General
              </button>
              <button 
                onClick={() => setActiveTab('prompt')}
                className={cn("px-3 py-1.5 text-xs font-medium border-b-2 transition-colors", activeTab === 'prompt' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground')}
              >
                Prompt & Tools
              </button>
              <button 
                onClick={() => setActiveTab('advanced')}
                className={cn("px-3 py-1.5 text-xs font-medium border-b-2 transition-colors", activeTab === 'advanced' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground')}
              >
                Advanced
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-5 text-sm">
              {activeTab === 'config' && (
                <>
                  <div className="space-y-1.5">
                    <label className="text-xs font-semibold text-foreground">Task Identifier</label>
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
                    <label className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                       <LayoutGrid size={14} className="text-muted-foreground" /> Provider Routing
                    </label>
                    <select 
                       value={(selectedNode.data.provider as string) || ''}
                       onChange={(e) => updateNodeData(selectedNode.id, { provider: e.target.value, model: '' })}
                       className="w-full bg-background border border-border rounded-md px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value="">Auto-Route (Recommended)</option>
                      {providers.map(p => (
                        <option key={p.name} value={p.name}>{p.name}</option>
                      ))}
                    </select>
                  </div>

                  {!!selectedNode.data.provider && (
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold text-foreground">Target Model</label>
                      <select 
                        value={(selectedNode.data.model as string) || ''}
                        onChange={(e) => updateNodeData(selectedNode.id, { model: e.target.value })}
                        className="w-full bg-background border border-border rounded-md px-2.5 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                      >
                        <option value="">Default for Provider</option>
                        {providers.find(p => p.name === selectedNode.data.provider)?.models.map(m => (
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
                      <label className="text-xs font-semibold text-foreground flex items-center justify-between">
                        System Prompt
                        <button className="text-[10px] text-primary hover:underline">Insert Variable</button>
                      </label>
                      <textarea 
                        className="w-full bg-secondary/30 font-mono text-xs border border-border rounded-md px-2.5 py-2 focus:outline-none focus:ring-1 focus:ring-primary h-32 resize-y"
                        placeholder="You are an expert orchestration agent..."
                      ></textarea>
                   </div>
                   
                   <div className="space-y-1.5">
                      <label className="text-xs font-semibold text-foreground">Available Tools</label>
                      <div className="border border-border rounded-md p-2 space-y-1 bg-background">
                         {['web_search', 'read_file', 'execute_code'].map(tool => (
                           <label key={tool} className="flex items-center gap-2 text-xs">
                             <input type="checkbox" className="rounded border-border text-primary focus:ring-primary" />
                             {tool}
                           </label>
                         ))}
                      </div>
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
                           <input type="number" defaultValue={3} className="w-full bg-background border border-border rounded-md px-2.5 py-1 text-xs" />
                        </div>
                        <div>
                           <label className="text-[10px] text-muted-foreground block mb-1">Timeout (ms)</label>
                           <input type="number" defaultValue={30000} className="w-full bg-background border border-border rounded-md px-2.5 py-1 text-xs" />
                        </div>
                      </div>
                   </div>

                   <div className="space-y-1.5 pt-4 border-t border-border">
                      <label className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                         <Key size={14} className="text-muted-foreground" /> Execution Constraints
                      </label>
                      <label className="flex items-center gap-2 text-xs mt-2">
                        <input type="checkbox" className="rounded border-border text-primary focus:ring-primary" />
                        Require Human Approval Before Execute
                      </label>
                      <label className="flex items-center gap-2 text-xs">
                        <input type="checkbox" className="rounded border-border text-primary focus:ring-primary" />
                        Cache Output (Deterministic)
                      </label>
                   </div>

                   <div className="pt-8">
                     <button 
                       onClick={() => setNodes(nds => nds.filter(n => n.id !== selectedNode.id))}
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
