import { create } from 'zustand';
import type { Workflow, Provider } from '@/types';
import api from '@/api/client';

interface WorkflowState {
  workflows: Workflow[];
  providers: Provider[];
  isLoading: boolean;
  error: string | null;

  fetchWorkflows: () => Promise<void>;
  fetchProviders: () => Promise<void>;
  createWorkflow: (workflow: Partial<Workflow>) => Promise<void>;
  deleteWorkflow: (id: string) => Promise<void>;
}

export const useWorkflowStore = create<WorkflowState>((set) => ({
  workflows: [],
  providers: [],
  isLoading: false,
  error: null,

  fetchWorkflows: async () => {
    set({ isLoading: true });
    try {
      const response = await api.get('/workflows');
      set({ workflows: response.data, isLoading: false });
    } catch (error: any) {
      set({ error: error.message, isLoading: false });
    }
  },

  fetchProviders: async () => {
    try {
      const response = await api.get('/meta/providers');
      set({ providers: response.data });
    } catch (error: any) {
      console.error('Failed to fetch providers', error);
    }
  },

  createWorkflow: async (workflow) => {
    set({ isLoading: true });
    try {
      await api.post('/workflows', workflow);
      const response = await api.get('/workflows');
      set({ workflows: response.data, isLoading: false });
    } catch (error: any) {
      set({ error: error.message, isLoading: false });
    }
  },

  deleteWorkflow: async (id) => {
    // Note: Backend doesn't have DELETE /workflows/{id} yet, but we'll add it if needed
    // For now just client-side or assume it exists
    try {
      // await api.delete(`/workflows/${id}`);
      set((state) => ({
        workflows: state.workflows.filter((w) => w.id !== id),
      }));
    } catch (error: any) {
      set({ error: error.message });
    }
  },
}));
