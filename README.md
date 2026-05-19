# LLM Orchestrator

LLM Orchestrator is a production-grade, local-first execution engine designed to transform declarative workflow definitions into robust, dynamic execution graphs (DAGs) for AI agents. 

It prioritizes reliability, modularity, and intelligence, handling complex task dependencies, multi-agent coordination, and resilient state persistence.

## Key Features

- **Dynamic DAG Execution:** Automatically manages task dependencies and parallel execution stages.
- **Multi-Agent Coordination:** Role-based routing with capability-aware and cost-aware logic.
- **Context Management:** 
    - **JSON Artifact Injection:** Automatically passes data between tasks in clean JSON format.
    - **Template Interpolation:** Reference previous task outputs using `{{TaskName.field}}`.
    - **Context Stitching:** Intelligently manages context window limits.
- **Human-in-the-Loop:** Pause execution for manual approval or intervention.
- **Resiliency:** Automatic retries, panic recovery, and checkpointing.
- **Observability:** Real-time event stream via WebSockets, token usage tracking, and structured audit logging.
- **Local-First:** Optimized for local development with optional PostgreSQL persistence.

## Installation

### Prerequisites
- **Go:** 1.22 or higher.
- **Node.js & npm:** For building the UI.
- **Docker:** (Optional) For containerized deployment.
- **API Keys:** At least one provider key (Groq, OpenAI, Anthropic, or Gemini).

### Setup
1.  **Clone the repository:**
    ```bash
    git clone https://github.com/AniruthKarthik/llm-orchestrator.git
    cd llm-orchestrator
    ```

2.  **Configure Environment Variables:**
    ```bash
    cp .env.example .env
    # Edit .env and add your API keys
    ```

3.  **Build and Run:**
    ```bash
    make build  # Builds the UI and the Go binary
    make run    # Starts the server at http://localhost:8080
    ```

## Usage

### Workflow Builder (UI)
The most intuitive way to use LLM Orchestrator is through the built-in Workflow Builder. It allows you to:
- Drag and drop tasks.
- Connect tasks to define dependencies.
- Configure agents, models, and prompts.
- Execute workflows and monitor real-time logs.

### CLI Tool (`orch`)
For quick local execution of YAML-defined workflows:
```bash
go run ./cmd/orch workflow.yaml
```

### API Server
The backend exposes a REST API for programmatic control.
- `POST /api/v1/workflows`: Create/Update workflows.
- `POST /api/v1/workflows/{id}/execute`: Trigger execution.
- `GET /api/v1/metrics`: View system performance and token usage.

## Documentation
- [Architecture Overview](architecture.md): Deep dive into the system design.
- [Environment Variables](#environment-variables): Configuration details.

## Environment Variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string (leave empty for in-memory). |
| `API_KEY` | Optional security key for the API. |
| `GROQ_API_KEY` | Your Groq API key. |
| `OPENAI_API_KEY` | Your OpenAI API key. |
| `ANTHROPIC_API_KEY` | Your Anthropic API key. |
| `GEMINI_API_KEY` | Your Google Gemini API key. |

## License
MIT
