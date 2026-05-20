# LLM Orchestrator: Comprehensive Technical Architecture

LLM Orchestrator is a robust, modular, and local-first execution engine designed to manage complex AI workflows. It represents workflows as Directed Acyclic Graphs (DAGs), where each node is a discrete unit of work (Task) and edges represent dependencies.

---

## 1. Core Philosophy & Design Principles

The system is built on three pillars:
- **Resilience:** Built-in retries, panic recovery, and state persistence ensure that workflows can recover from failures at any stage.
- **Modularity:** A plugin-based architecture and a clean separation of concerns allow for easy extension of providers, agents, and execution logic.
- **Transparency:** Comprehensive observability through an asynchronous event bus, real-time WebSocket updates, and structured audit logging.

---

## 2. Domain Model & Primitives (`internal/core/`)

### 2.1 Workflow
A `Workflow` is the top-level container for a set of tasks.
- **State Machine:** Transitions from `PENDING` -> `RUNNING` -> `COMPLETED` or `FAILED`.
- **Failure Policies:**
    - `FailFast`: Immediately halts the workflow if any task fails.
    - `ContinueOnFailure`: Continues executing independent tasks even if some fail.
- **Thread Safety:** Uses `sync.RWMutex` to protect access to task maps and status fields.

### 2.2 Task
The fundamental unit of execution.
- **Input/Output:** Raw data maps (`map[string]any`) that are deep-copied to ensure data integrity during parallel execution.
- **Status Lifecycle:** `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, and `WAITING_FOR_APPROVAL`.
- **Output Validation:** Supports an `OutputSchema` (e.g., `{"summary": "string", "count": "int"}`) which is enforced after execution to ensure LLM outputs meet structural requirements.
- **Human-in-the-Loop:** Tasks can be flagged with `RequiresApproval`, causing them to pause and wait for external signal (via REST API) before continuing.

### 2.3 Artifacts
Artifacts are versioned, immutable records of data produced by tasks.
- **Automated Registration:** The executor automatically captures task outputs as artifacts.
- **Context Injection:** Downstream tasks can reference artifacts by name or ID, which are then injected into the agent's context.

### 2.4 Tools & Memory
- **Tools:** Discrete functions or external APIs that agents can call during execution.
- **Memory:** Short-term and long-term storage for agents, allowing for stateful conversations across workflow runs.

---

## 3. DAG Engine & Execution Planning (`internal/dag/`)

The Orchestrator uses a multi-step process to transform a declarative workflow into an execution plan.

### 3.1 Validation
The `Validator` ensures the graph is logically sound:
- **Reference Check:** Ensures all task dependencies exist within the workflow.
- **Cycle Detection:** Uses a Depth-First Search (DFS) with a recursion stack to detect circular dependencies, preventing infinite loops.
- **Self-Dependency Check:** Prevents tasks from depending on themselves.

### 3.2 Topological Planning
The `TopologicalPlanner` uses Kahn's algorithm to organize tasks into execution stages:
1.  **Indegree Calculation:** Counts the number of incoming dependencies for each task.
2.  **Stage Generation:** Tasks with an indegree of 0 are grouped into the current stage.
3.  **Iteration:** For each task in the current stage, the indegree of its neighbors is decremented. Tasks that hit an indegree of 0 are added to the next stage.
4.  **Parallelism:** All tasks within a single stage can be executed concurrently.

---

## 4. Execution Engine (`internal/executor/`)

The `Executor` is the heart of the system, responsible for orchestrating the lifecycle of a workflow.

### 4.1 Orchestration Loop
1.  **Preparation:** Saves the initial workflow and task states to the `Store`.
2.  **Planning:** Invokes the DAG engine to generate an `ExecutionPlan`.
3.  **Stage-by-Stage Execution:**
    - Tasks within a stage are spawned in separate goroutines.
    - A `sync.WaitGroup` manages synchronization for the stage.
    - A concurrency-limited worker pool (semaphore-based) prevents system exhaustion.
4.  **Checkpointing:** After each stage, the system serializes the entire workflow state and saves a `Checkpoint`. This allows for `Resume()` functionality after a crash or manual pause.

### 4.2 Middleware Pipeline
Execution logic is wrapped in two layers of middleware:
- **Workflow Middleware:** Wraps the entire workflow execution (e.g., logging, global timeouts).
- **Task Middleware:** Wraps individual task execution (e.g., retries, output validation, metric collection).

### 4.3 Routing
If a task isn't assigned a specific agent, the `Router` dynamically selects one:
- **Capability-Aware Routing:** Matches task requirements against agent roles and descriptions.
- **Cost-Aware Routing:** Selects models based on a predefined cost registry to minimize token expenditure.

---

## 5. Intelligence & Agents (`internal/agents/`)

### 5.1 Agent Registry
A central repository of agent definitions, including their roles, system prompts, and tool access permissions.

### 5.2 Context Engineering
Before an LLM call, the `AgentExecutor` constructs the final prompt:
- **Artifact Injection:** Serializes relevant artifacts into structured JSON and prepends them to the system prompt.
- **Template Interpolation:** Uses regex-based interpolation to replace `{{TaskName.field}}` placeholders with actual values from upstream task outputs.
- **Context Stitching:** The `ContextStitcher` ensures the combined prompt fits within the model's context window. It uses token-counting logic to intelligently truncate or omit less relevant data while maintaining the "thread" of the workflow.

---

## 6. Provider Abstraction (`internal/providers/`)

The system implements a unified interface for multiple LLM providers (Groq, OpenAI, Gemini, Anthropic).

- **Unified Request/Response:** All provider-specific details are mapped to a common `GenerateRequest` and `GenerateResponse` structure.
- **Mappers:** Each provider has a specialized mapper to translate system/user prompts and tool definitions into the provider's specific API format.
- **Registry:** A thread-safe registry allows for dynamic lookup and instantiation of providers based on model names.

---

## 7. Persistence & State (`internal/store/`)

The system follows the **Repository Pattern** to decouple domain logic from storage implementations.

- **Storage Interfaces:** Defines `Store`, `WorkflowStore`, `TaskStore`, and `ArtifactStore`.
- **Implementations:**
    - **Memory Store:** Uses thread-safe maps for local development and testing.
    - **Postgres Store:** Uses `sqlx` for production-grade persistence, supporting ACID transactions and complex queries.
- **Object Mapping:** Domain objects are converted to "Record" structs (e.g., `TaskRecord`) before being persisted to ensure database schema changes don't leak into core logic.

---

## 8. Async Event Bus (`internal/events/`)

A high-performance, asynchronous pub/sub system facilitates decoupling between components.

- **Buffered Channels:** Events are published to a buffered channel to prevent blocking the main execution thread.
- **Worker Pool:** A pool of background workers dispatches events to registered subscribers.
- **Event Types:** Includes `WorkflowStarted`, `TaskCompleted`, `TaskFailed`, `StageStarted`, `TaskTokenUsage`, etc.

---

## 9. Observability & API (`internal/api/` & `internal/observer/`)

### 9.1 REST API
Provides endpoints for workflow management, execution triggering, and system configuration.

### 9.2 Real-time Synchronization (WebSockets)
A WebSocket hub subscribes to the `EventBus` and broadcasts events to connected UI clients. This allows the Workflow Builder to show live status updates (e.g., a node turning green when a task completes) without polling.

### 9.3 Observer Package
- **Metrics:** Tracks token usage, latency, and success rates using Prometheus-compatible collectors.
- **Audit Logging:** Records every significant action and state change into a structured log for compliance and debugging.

---

## 10. Frontend Architecture (`ui/`)

The UI is a modern React application built with TypeScript and Vite.

- **State Management:** Uses `Zustand` for lightweight, reactive state management of workflows and system configuration.
- **Workflow Builder:** A drag-and-drop interface powered by `React Flow` for visualizing and designing DAGs.
- **Real-time Updates:** Integrates with the backend WebSocket to provide a "live" execution dashboard.
- **UI Components:** Built with a custom design system using `Vanilla CSS` for maximum performance and flexibility.

---

## 11. Directory Structure

```text
├── cmd/
│   ├── orch/          # CLI tool for executing YAML workflows
│   └── server/        # Main API server and UI host
├── internal/
│   ├── agents/        # Agent definitions, registries, and executors
│   ├── api/           # REST handlers and WebSocket Hub
│   ├── core/          # Domain Logic: Workflow, Task, Artifact, Tools, Memory
│   ├── dag/           # Planning: Cycle detection, Topological sort
│   ├── dsl/           # Domain Specific Language for YAML parsing
│   ├── events/        # Async Event Bus (Pub/Sub)
│   ├── executor/      # Execution Engine: Concurrency, Middleware, Checkpointing
│   ├── providers/     # LLM Adapters: OpenAI, Groq, Gemini, Anthropic
│   ├── store/         # Persistence: Postgres and Memory implementations
│   └── observer/      # Observability: Metrics, Auditing, Logging
└── ui/
    ├── src/components # Reusable UI primitives and Task nodes
    ├── src/pages      # Main views: Builder, Dashboard, Executions
    └── src/store      # Zustand state stores
```
