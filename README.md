# LLM Orchestrator

LLM Orchestrator is a production-grade, local-first execution engine designed to transform declarative workflow definitions into robust, dynamic execution graphs (DAGs) for AI agents. 

It prioritizes reliability, modularity, and intelligence, handling complex task dependencies, multi-agent coordination, and resilient state persistence.

---

## Key Features

- **Dynamic DAG Execution:** Automatically manages task dependencies and parallel execution stages using topological sorting.
- **Multi-Agent Coordination:** Role-based routing with capability-aware and cost-aware logic.
- **Advanced Context Management:** 
    - **JSON Artifact Injection:** Automatically passes data between tasks in clean JSON format.
    - **Template Interpolation:** Reference previous task outputs using `{{TaskName.field}}`.
    - **Context Stitching:** Intelligently manages context window limits with token-aware truncation.
- **Human-in-the-Loop:** Pause execution for manual approval or intervention via REST/UI.
- **Resiliency:** Automatic retries, panic recovery, and stage-level checkpointing.
- **Real-time Observability:** Event-driven architecture with WebSockets for live UI updates and token usage tracking.
- **Pluggable Providers:** Native support for Groq, OpenAI, Anthropic, and Google Gemini.
- **Persistence:** Repository pattern with support for in-memory and PostgreSQL storage.

---

## Architecture at a Glance

The system is composed of several decoupled layers:

1.  **Core Domain:** Defines the `Workflow`, `Task`, and `Artifact` primitives.
2.  **DAG Engine:** Validates graphs and computes the optimal topological execution order.
3.  **Executor:** Manages the lifecycle of workflow runs, handling concurrency, retries, and middleware.
4.  **Agent Layer:** Handles prompt engineering, context injection, and agent-specific execution logic.
5.  **Provider Layer:** Unified interface for interacting with various LLM APIs.
6.  **Persistence Layer:** Abstracts state management across different storage engines.
7.  **API & UI:** Provides a RESTful interface and a reactive React-based workflow builder.

For a deep dive, see the [Architecture Documentation](architecture.md).

---

## Getting Started

### Prerequisites
- **Go:** 1.22+
- **Node.js & npm:** For the frontend UI.
- **PostgreSQL:** (Optional) For persistent storage.
- **API Keys:** At least one key from Groq, OpenAI, Anthropic, or Gemini.

### Installation

1.  **Clone and Enter:**
    ```bash
    git clone https://github.com/AniruthKarthik/llm-orchestrator.git
    cd llm-orchestrator
    ```

2.  **Configuration:**
    ```bash
    cp .env.example .env
    # Edit .env and add your provider API keys
    ```

3.  **Build and Run:**
    ```bash
    make build  # Builds UI and Go binaries
    make run    # Starts the server at http://localhost:8080
    ```

---

## Usage

### Workflow Builder (UI)
The primary interface for designing and monitoring workflows.
- **Visual Design:** Drag and drop tasks and connect them to define dependencies.
- **Live Monitoring:** Watch execution progress in real-time with node status changes.
- **Agent Config:** Define agent roles, system prompts, and model assignments.

### CLI Tool (`orch`)
For automated or headless execution of YAML-defined workflows:
```bash
./bin/orch examples/research_workflow.yaml
```

### API
The backend exposes a full REST API for programmatic integration:
- `POST /api/v1/workflows`: Create or update workflow definitions.
- `POST /api/v1/workflows/{id}/execute`: Trigger a workflow run.
- `GET /api/v1/executions`: List recent workflow executions and their status.

---

## Testing

The project maintains a high test coverage across core packages:
```bash
go test ./internal/... -v
```

---

## License
This project is licensed under the MIT License - see the LICENSE file for details.
