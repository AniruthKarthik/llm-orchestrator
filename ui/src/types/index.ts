export interface Workflow {
  id: string;
  name: string;
  description: string;
  status: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  taskCount?: number;            // populated on list endpoint
  tasks?: Record<string, Task>;  // populated on GET /workflows/:id
}

export interface Task {
  id: string;
  workflowId: string;
  name: string;
  description: string;
  status: string;
  error?: string;
  input: Record<string, unknown>;
  output?: Record<string, unknown>;
  dependencies: string[];
  timeout?: number;
  retryPolicy?: {
    MaxRetries?: number;
    maxRetries?: number;
  };
  attempt?: number;
  agentId?: string;
  provider?: string;
  model?: string;
  requiresApproval?: boolean;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface Provider {
  name: string;
  models: string[];
}

// Agent as returned by GET /agents (store.AgentRecord shape — camelCase)
export interface Agent {
  id: string;
  name: string;
  description: string;
  role: string;
  systemPrompt?: string;
  model: string;
  provider: string;
  tools?: string[];
  config?: Record<string, unknown>;
}

// Artifact as returned by GET /artifacts (store.ArtifactRecord shape — camelCase)
export interface Artifact {
  id: string;
  workflowId: string;
  taskId: string;
  name: string;
  type: string;
  data: unknown;
  metadata?: Record<string, unknown>;
  createdAt: string;
}

export interface Event {
  type: string;
  payload: unknown;
  timestamp: string;
}

export type TaskStatus =
  | 'PENDING'
  | 'RUNNING'
  | 'COMPLETED'
  | 'FAILED'
  | 'WAITING_FOR_APPROVAL';

export type WorkflowStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED';
