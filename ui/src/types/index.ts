export interface Workflow {
  id: string;
  name: string;
  description: string;
  status: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  tasks?: Record<string, Task>;
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

// Agent as returned by GET /agents (store.AgentRecord shape)
export interface Agent {
  ID: string;
  Name: string;
  Description: string;
  Role: string;
  Model: string;
  Provider: string;
  Tools: string[];
  Config: Record<string, unknown>;
}

// Artifact as returned by GET /artifacts (store.ArtifactRecord shape)
export interface Artifact {
  ID: string;
  WorkflowID: string;
  TaskID: string;
  Name: string;
  Type: string;
  Data: unknown;
  Metadata: Record<string, unknown>;
  CreatedAt: string;
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
