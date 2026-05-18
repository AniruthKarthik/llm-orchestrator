export interface Workflow {
  id: string;
  name: string;
  description: string;
  status: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  tasks?: Task[];
}

export interface Task {
  id: string;
  workflowId: string;
  name: string;
  description: string;
  status: string;
  error?: string;
  input: Record<string, any>;
  output?: Record<string, any>;
  dependencies: string[];
  agentId?: string;
  provider?: string;
  model?: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface Provider {
  name: string;
  models: string[];
}

export interface Event {
  type: string;
  payload: any;
  timestamp: string;
}

export type TaskStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED' | 'WAITING_FOR_APPROVAL';
