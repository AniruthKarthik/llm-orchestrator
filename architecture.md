# LLM Orchestrator: System Architecture

## 1. Overview
The LLM Orchestrator is a production-grade, local-first execution engine for AI workflows. It transforms declarative definitions (YAML/JSON) into dynamic execution graphs (DAGs), managing complex dependencies, agent assignments, and state persistence with high reliability.

## 2. Core Components

### 2.1 Engine & Execution
*   **Executor:** The central brain that orchestrates task execution. It manages concurrency, applies middleware (retry, recovery, validation), and coordinates between workers and agents.
*   **Supervisor:** A background watcher that detects "stuck" tasks or non-responsive execution paths, triggering recovery or failure policies.
*   **Worker Registry:** A pluggable registry for local tool execution (Shell, Python, Search).
*   **Agent Registry:** Manages agent definitions, mapping roles to specific LLM models and providers.

### 2.2 Intelligence & Routing
*   **Dynamic Router:** Decides at runtime which agent or model should handle a task.
    *   **Capability-Aware:** Matches tasks to agents based on the agent's role and capabilities.
    *   **Cost-Aware:** Optimizes for budget by selecting the cheapest model that meets requirements.
    *   **Fallback Logic:** Automatically switches to secondary providers (e.g., from Groq to OpenAI) if the primary provider is unavailable.
*   **Context Management:**
    *   **Retrieval Injection:** Automatically searches and injects relevant artifacts from previous tasks into the LLM context.
    *   **Context Stitching:** Manages the context window by intelligently chunking and summarizing voluminous data before sending it to providers.

### 2.3 Persistence Layer
*   **SQL Store (PostgreSQL):** Persists the entire state of workflows and tasks, allowing for resume/recovery after crashes.
*   **Memory Store:** A high-performance, in-memory implementation for transient testing and low-latency local execution.
*   **Checkpointing:** Periodically saves the full execution state to allow resuming long-running workflows from the last successful stage.

## 3. The Execution Lifecycle

1.  **DSL Compilation:** A YAML/JSON workflow definition is parsed and validated (cycle detection, dependency checks).
2.  **DAG Generation:** The compiler produces a directed acyclic graph of tasks.
3.  **Topological Planning:** The DAG is decomposed into execution stages (groups of tasks that can run in parallel).
4.  **Task Execution:** For each task:
    *   The **Router** selects an agent.
    *   The **AgentExecutor** performs **Retrieval Injection**.
    *   The LLM provider generates a response.
    *   **Output Validation** middleware ensures the response matches the defined schema.
    *   **Artifacts** are produced and saved to the registry.
5.  **Completion:** The workflow state is updated, and metrics are finalized.

## 4. Human-in-the-loop (HITL)
The system supports manual intervention through **Approval Checkpoints**.
*   Tasks can be marked with `requires_approval: true`.
*   The Executor pauses execution at these tasks, moving them to a `WAITING_FOR_APPROVAL` state.
*   An external signal (via REST API) is required to resume execution.

## 5. Observability & Security

*   **Event Bus:** A high-throughput, internal pub/sub system that broadcasts state changes and metrics.
*   **Audit Logger:** Captures an immutable, chronologically ordered log of all system events, model interactions, and human approvals.
*   **Metrics Collector:** Tracks token usage, model costs, task latency, and failure rates (compatible with Prometheus).
*   **Secret Management:** Securely retrieves API keys and credentials from environment variables, ensuring zero-exposure in logs or persistence.

## 6. Directory Structure
```text
├── cmd/               # Entry points (orch CLI, API Server)
├── internal/
│   ├── core/          # Domain models (Workflow, Task, Artifact)
│   ├── executor/      # Execution engine and Routing logic
│   ├── agents/        # Agent definitions and Context management
│   ├── dsl/           # YAML/JSON parsing and Validation
│   ├── providers/     # LLM Provider integrations (Groq, OpenAI, etc.)
│   ├── store/         # Persistence (Postgres, Memory)
│   ├── events/        # Internal Event Bus
│   └── observer/      # Metrics and Audit Logging
└── migrations/        # Database schema migrations
```
