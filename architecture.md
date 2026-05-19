# LLM Orchestrator: Technical Architecture Deep Dive

## 1. System Philosophy
LLM Orchestrator is built as a **modular, local-first execution engine**. It treats AI workflows as Directed Acyclic Graphs (DAGs) where each node is a discrete unit of work (Task) and each edge is a dependency. 

The core design goals are:
- **Deterministic Scheduling:** Parallel execution whenever possible, serial when necessary.
- **State Reliability:** Persistence-first approach to allow recovery from any point of failure.
- **Contextual Intelligence:** Ensuring agents have structured, relevant data from upstream tasks.

---

## 2. Domain Model & Core Primitives

### 2.1 Workflow (`internal/core/workflow.go`)
A `Workflow` is the top-level container. It maintains:
- **Registry of Tasks:** A thread-safe map (`map[string]*Task`) protected by an `RWMutex`.
- **Status Lifecycle:** `PENDING` -> `RUNNING` -> (`COMPLETED` | `FAILED`).
- **Failure Policy:** Determines if the entire workflow halts (`FailFast`) or continues (`ContinueOnFailure`) when a task fails.

### 2.2 Task (`internal/core/task.go`)
The fundamental unit of execution. Key attributes include:
- **Input/Output:** Raw data maps (`map[string]any`) that are deep-copied during transitions to prevent race conditions.
- **Dependencies:** A list of Task IDs that must reach `COMPLETED` before this task can start.
- **Output Schema:** A type-validation map (e.g., `{"count": "int"}`) used by middleware to verify LLM responses.
- **Retry Policy:** Configurable `MaxRetries` and backoff strategies (e.g., `Immediate`).

### 2.3 Artifact (`internal/core/artifact.go`)
Long-lived data produced by tasks. Artifacts are automatically registered upon task completion and serve as the primary source for context injection in downstream tasks.

---

## 3. DAG Engine & Planning (`internal/dag/`)

The Orchestrator doesn't just "run" tasks; it compiles them into an **Execution Plan**.

1.  **Validation:** The `Validator` checks for cycles (using DFS), missing dependencies, and duplicate IDs.
2.  **Topological Sort:** The `TopologicalPlanner` uses Kahn's algorithm:
    - **Indegree Map:** Counts how many dependencies each task has.
    - **Adjacency List:** Maps each task to the tasks that depend on it.
    - **Stage Generation:** Tasks with 0 indegree are grouped into `Stage 0`. They are marked as processed, and their neighbors' indegrees are decremented. Tasks that hit 0 indegree in the next iteration form `Stage 1`, and so on.

---

## 4. The Execution Engine (`internal/executor/`)

### 4.1 Executor Workflow
The `Executor` processes the `ExecutionPlan` stage-by-stage. Within a stage, tasks are executed concurrently using a `sync.WaitGroup`.

### 4.2 Middleware Pipeline
Task execution is wrapped in a middleware chain (`ApplyTaskMiddleware`):
- **Output Validation:** Intercepts the result and matches it against the `OutputSchema`.
- **Interceptors:** Hooks for observers to record metrics or logs before/after execution.

### 4.3 Resource Management
- **Concurrency Limiter:** A semaphore-based (`chan struct{}`) limiter ensures the system doesn't overwhelm the host machine or LLM rate limits.
- **Token Bucket Limiter:** Implements rate-limiting for outgoing provider requests to handle model-specific quotas.

---

## 5. Intelligence & Context Engineering

### 5.1 Dynamic Routing (`internal/executor/router.go`)
Routers decide which agent handles a task if one isn't explicitly assigned:
- **CapabilityAwareRouter:** Analyzes agent roles (Researcher, Executor, etc.) against task metadata.
- **CostAwareRouter:** Uses a `ModelCostRegistry` to select the most economical model for the requested role.

### 5.2 Context Injection Engine (`internal/agents/agents.go`)
Before an agent receives a task, the `AgentExecutor` performs two critical data-passing steps:
1.  **JSON Artifact Injection:** All artifacts from the same `WorkflowID` are fetched. Their data is serialized to structured JSON and injected into the System Prompt. This allows the LLM to "see" the history of the workflow.
2.  **Template Interpolation:** The user prompt is scanned for placeholders like `{{TaskName.field}}`. These are replaced with actual values from upstream task outputs before the request is sent to the LLM.

### 5.3 Context Stitching (`internal/agents/stitching.go`)
To prevent "Context Window Overflow," the `ContextStitcher` monitors the character count of injected data. It intelligently truncates or omits older/less relevant artifacts while preserving the workflow's structural context.

---

## 6. Persistence & State Management (`internal/store/`)

The system implements a **Repository Pattern** to abstract the storage engine.

- **Record Mapping:** Domain objects (like `core.Task`) are converted to `store.TaskRecord` (plain structs with JSON tags) before being saved.
- **Postgres Store:** Uses `database/sql` for ACID-compliant persistence. Every state change (Task Start, Task Complete, Workflow Fail) is an atomic update to the DB.
- **Checkpointing:** At the end of every `ExecutionStage`, a full snapshot of the workflow state is serialized and saved as a Checkpoint. This allows the engine to `Resume()` a workflow even after a full system restart.

---

## 7. Observability & Communication

### 7.1 Internal Event Bus (`internal/events/`)
A central pub/sub system (`EventBus`) allows internal components to stay decoupled.
- **Publishers:** The Executor publishes events like `TaskStarted`, `TaskTokenUsage`, `WorkflowCompleted`.
- **Subscribers:** The Audit Logger (for file persistence), the Metrics Collector (for Prometheus), and the API Server (for real-time UI updates).

### 7.2 Real-time UI Sync
The API Server maintains a WebSocket (`/api/v1/ws`) that subscribes to the `EventBus`. When an event occurs in the Go backend, it is immediately pushed as JSON to the React frontend, allowing the Workflow Builder to show live node status changes without polling.

---

## 8. Directory Architecture Summary

```text
├── cmd/
│   ├── orch/          # CLI tool for YAML execution
│   └── server/        # Main API & UI server
├── internal/
│   ├── core/          # Domain Logic: Workflow, Task, Artifact, Retry
│   ├── dag/           # Planning: Cycle detection, Topological sort
│   ├── executor/      # Execution: Concurrency, Middleware, Routing, Checkpoints
│   ├── agents/        # Intelligence: Prompt engineering, Context injection, Stitching
│   ├── providers/     # LLM Adapters: Groq, OpenAI, Gemini, Anthropic
│   ├── store/         # Persistence: Postgres/Memory implementations
│   ├── events/        # Communication: Pub/Sub Event Bus
│   ├── api/           # Interface: REST handlers and WebSocket Hub
│   └── observer/      # Insights: Logging, Metrics, Tracing
└── ui/
    ├── src/components # UI Primitives (TaskNodes, Layouts)
    ├── src/pages      # Workflow Builder and Dashboard logic
    └── src/store      # State management (Zustand)
```
