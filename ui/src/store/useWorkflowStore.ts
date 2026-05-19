import { create } from 'zustand';
import type { Workflow, Provider, Agent, Artifact, Task } from '@/types';
import api from '@/api/client';
import { toast } from '@/store/useToastStore';

// Per-resource slice to avoid cross-page loading/error state bleed
interface ResourceState<T> {
  data: T[];
  isLoading: boolean;
  error: string | null;
}

interface MetricsData {
  activeWorkflows: number;
  completedWorkflows: number;
  failedWorkflows: number;
  pendingWorkflows: number;
  totalWorkflows: number;
  tasksInQueue: number;
  providersOnline: number;
  providers: Provider[];
}

interface WorkflowState {
  // Per-resource slices
  workflows: ResourceState<Workflow>;
  providers: ResourceState<Provider>;
  agents: ResourceState<Agent>;
  artifacts: ResourceState<Artifact>;
  queueTasks: ResourceState<Task>;
  metrics: MetricsData | null;
  metricsLoading: boolean;

  // Actions
  fetchWorkflows: () => Promise<void>;
  fetchProviders: () => Promise<void>;
  fetchAgents: () => Promise<void>;
  fetchArtifacts: () => Promise<void>;
  fetchQueues: () => Promise<void>;
  fetchMetrics: () => Promise<void>;
  createWorkflow: (workflow: Record<string, unknown>) => Promise<Workflow | null>;
  deleteWorkflow: (id: string) => Promise<boolean>;
  executeWorkflow: (id: string, options?: { dryRun?: boolean }) => Promise<boolean>;
}

const emptyResource = <T>(): ResourceState<T> => ({
  data: [],
  isLoading: false,
  error: null,
});

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  workflows: emptyResource<Workflow>(),
  providers: emptyResource<Provider>(),
  agents: emptyResource<Agent>(),
  artifacts: emptyResource<Artifact>(),
  queueTasks: emptyResource<Task>(),
  metrics: null,
  metricsLoading: false,

  fetchWorkflows: async () => {
    set((s) => ({ workflows: { ...s.workflows, isLoading: true, error: null } }));
    try {
      const response = await api.get('/workflows');
      set({ workflows: { data: response.data ?? [], isLoading: false, error: null } });
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ workflows: { ...s.workflows, isLoading: false, error: msg } }));
    }
  },

  fetchProviders: async () => {
    set((s) => ({ providers: { ...s.providers, isLoading: true, error: null } }));
    try {
      const response = await api.get('/meta/providers');
      set({ providers: { data: response.data ?? [], isLoading: false, error: null } });
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ providers: { ...s.providers, isLoading: false, error: msg } }));
    }
  },

  fetchAgents: async () => {
    set((s) => ({ agents: { ...s.agents, isLoading: true, error: null } }));
    try {
      const response = await api.get('/agents');
      set({ agents: { data: response.data ?? [], isLoading: false, error: null } });
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ agents: { ...s.agents, isLoading: false, error: msg } }));
    }
  },

  fetchArtifacts: async () => {
    set((s) => ({ artifacts: { ...s.artifacts, isLoading: true, error: null } }));
    try {
      const response = await api.get('/artifacts');
      set({ artifacts: { data: response.data ?? [], isLoading: false, error: null } });
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ artifacts: { ...s.artifacts, isLoading: false, error: msg } }));
    }
  },

  fetchQueues: async () => {
    set((s) => ({ queueTasks: { ...s.queueTasks, isLoading: true, error: null } }));
    try {
      const response = await api.get('/queues');
      set({ queueTasks: { data: response.data ?? [], isLoading: false, error: null } });
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ queueTasks: { ...s.queueTasks, isLoading: false, error: msg } }));
    }
  },

  fetchMetrics: async () => {
    set({ metricsLoading: true });
    try {
      const response = await api.get('/metrics');
      set({ metrics: response.data, metricsLoading: false });
    } catch {
      set({ metricsLoading: false });
    }
  },

  createWorkflow: async (workflow) => {
    set((s) => ({ workflows: { ...s.workflows, isLoading: true, error: null } }));
    try {
      const response = await api.post('/workflows', workflow);
      // Refresh list after creation
      get().fetchWorkflows();
      // Only import toast inside the module to avoid circular deps if any
      const { toast } = await import('@/store/useToastStore');
      toast.success(`Workflow created successfully!`);
      return response.data as Workflow;
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ workflows: { ...s.workflows, isLoading: false, error: msg } }));
      return null;
    }
  },

  deleteWorkflow: async (id) => {
    try {
      await api.delete(`/workflows/${id}`);
      set((s) => ({
        workflows: {
          ...s.workflows,
          data: s.workflows.data.filter((w) => w.id !== id),
        },
      }));
      toast.success('Workflow deleted successfully.');
      return true;
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ workflows: { ...s.workflows, error: msg } }));
      toast.error(`Delete failed: ${msg}`);
      return false;
    }
  },

  executeWorkflow: async (id, options) => {
    try {
      await api.post(`/workflows/${id}/execute`, options || {});
      toast.success(options?.dryRun ? 'Dry-run simulation started.' : 'Workflow execution started.');
      // Refresh to get updated status
      setTimeout(() => get().fetchWorkflows(), 1000);
      return true;
    } catch (error: unknown) {
      const msg = extractErrorMessage(error);
      set((s) => ({ workflows: { ...s.workflows, error: msg } }));
      toast.error(`Execution failed: ${msg}`);
      return false;
    }
  },
}));

function extractErrorMessage(error: unknown): string {
  if (error && typeof error === 'object') {
    const err = error as Record<string, unknown>;
    // Axios error shape
    if (err.response && typeof err.response === 'object') {
      const resp = err.response as Record<string, unknown>;
      if (resp.data && typeof resp.data === 'object') {
        const data = resp.data as Record<string, unknown>;
        if (typeof data.error === 'string') return data.error;
      }
      if (typeof resp.data === 'string') return resp.data;
    }
    if (typeof err.message === 'string') return err.message;
  }
  return 'An unknown error occurred';
}
