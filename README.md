# LLM Orchestrator

The LLM Orchestrator is a production-grade, local-first execution engine designed to transform declarative workflow definitions into robust, dynamic execution graphs (DAGs) for AI agents. It prioritizes reliability and intelligence by providing a modular framework that handles complex task dependencies, multi-agent coordination with role-based routing (capability and cost-aware), and resilient state persistence via PostgreSQL. Built with a focus on local execution, it features advanced context management like retrieval injection and context stitching, ensuring that agents have the most relevant information while staying within model limits, all while maintaining an immutable audit trail and real-time observability.

## Installation

### Prerequisites
- **Go:** 1.25 or higher.
- **PostgreSQL:** (Optional) If you want persistent storage. You can also use Docker.
- **API Keys:** You will need API keys for at least one provider (e.g., Groq, OpenAI, Anthropic, or Gemini).

### Setup
1.  **Clone the repository:**
    ```bash
    git clone https://github.com/AniruthKarthik/llm-orchestrator.git
    cd llm-orchestrator
    ```

2.  **Configure Environment Variables:**
    Copy the example environment file and fill in your keys:
    ```bash
    cp .env.example .env
    ```

3.  **Run Database Migrations (if using Postgres):**
    The application handles migrations automatically on startup if a `DATABASE_URL` is provided.

## Usage

### 1. Running Workflows via CLI (`orch`)
The `orch` command is the fastest way to run a local workflow.

```bash
cp workflow.yaml.example workflow.yaml
# Edit workflow.yaml as needed
go run ./cmd/orch workflow.yaml
```

### 2. Running the API Server
The API server allows you to manage and trigger workflows programmatically.

```bash
# Start the server
go run ./cmd/server/main.go
```

**Common API Endpoints:**
- `POST /workflows`: Create a new workflow definition.
- `GET /workflows/{id}`: Retrieve workflow status and task outputs.
- `POST /workflows/{id}/execute`: Trigger the execution of a workflow.
- `POST /workflows/{id}/tasks/{taskID}/approve`: Approve a task that is waiting for human intervention.

### 3. Using Docker Compose
The easiest way to set up the entire stack (Postgres + Server) is using Docker Compose.

```bash
cp docker-compose.yaml.example docker-compose.yaml
docker-compose up --build
```

### 4. Defining a Workflow
Workflows are defined in YAML. Here is a simple example (`workflow.yaml`):

```yaml
id: research-summary-v1
name: "AI Research & Summary Workflow"
description: "A multi-stage workflow that researches a topic and waits for approval before summarizing."
tasks:
  - id: research-task
    name: "research-task"
    description: "Gather information about a specific topic"
    agent_id: "researcher-1"
    input:
      prompt: "Research the current state of local-first LLM orchestration."
    output_schema:
      research_notes: "string"

  - id: summary-task
    name: "summary-task"
    description: "Summarize the research notes"
    dependencies: ["research-task"]
    requires_approval: true # This task will wait for human approval
    agent_id: "comedian-1"
    input:
      prompt: "Based on the research notes, provide a concise summary."
```

- **Dynamic Routing:** Automatically selects the best agent based on role, cost, or availability.
- **Context Management:** Injects relevant artifacts from previous tasks into subsequent prompts.
- **Human-in-the-Loop:** Pause execution for manual approval or intervention.
- **Observability:** Built-in metrics for token usage, cost tracking, and structured JSON audit logging.
- **Resiliency:** Automatic retries, panic recovery, and checkpointing for long-running processes.
- **Security:** Optional API key authentication, configurable CORS, and 1 MiB request body limit.

---

## Developer Quick Start

```bash
cp .env.example .env        # Fill in at least one provider API key
make build                  # Compile backend binary + UI dist
make run                    # Start the server (backend serves UI at http://localhost:8080)

# Or run in dev mode (backend hot-reload + Vite HMR):
make run-dev                # Starts Air + Vite dev server (requires Air: go install github.com/air-verse/air@latest)
```

All available commands:

```
make help
```

## Production Deployment

```bash
# 1. Configure environment
cp .env.example .env && nano .env   # Set DATABASE_URL, provider keys, API_KEY, ALLOWED_ORIGIN

# 2. Docker Compose (recommended)
cp docker-compose.yaml.example docker-compose.yaml
docker-compose up --build -d

# 3. Or build and run the Docker image directly
make docker-build IMAGE=llm-orchestrator TAG=v1.0.0
make docker-run   IMAGE=llm-orchestrator TAG=v1.0.0
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | _(empty)_ | Postgres connection string. Empty = in-memory (data lost on restart). |
| `MIGRATIONS_PATH` | `migrations` | Path to SQL migration files. |
| `SERVER_PORT` | `:8080` | Bind port for the HTTP server. |
| `LOG_LEVEL` | `info` | Log verbosity: `info` or `debug`. |
| `ALLOWED_ORIGIN` | `*` | CORS allowed origin. Set to your UI domain in production. |
| `API_KEY` | _(empty)_ | If set, all API requests must include `X-API-Key` or `Authorization: Bearer` header. |
| `GROQ_API_KEY` | — | Groq provider API key. |
| `OPENAI_API_KEY` | — | OpenAI provider API key. |
| `ANTHROPIC_API_KEY` | — | Anthropic provider API key. |
| `GEMINI_API_KEY` | — | Google Gemini provider API key. |

## Full API Reference

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check (always public) |
| GET | `/api/v1/meta/providers` | List configured providers and their models |
| GET | `/api/v1/workflows` | List all workflows |
| POST | `/api/v1/workflows` | Create a workflow |
| GET | `/api/v1/workflows/{id}` | Get workflow detail + tasks |
| PUT | `/api/v1/workflows/{id}` | Update workflow definition |
| DELETE | `/api/v1/workflows/{id}` | Delete workflow (cascades tasks) |
| POST | `/api/v1/workflows/{id}/execute` | Start workflow execution |
| POST | `/api/v1/workflows/{id}/tasks/{taskID}/approve` | Approve a task awaiting human review |
| GET | `/api/v1/agents` | List registered agents |
| GET | `/api/v1/artifacts` | List all generated artifacts |
| GET | `/api/v1/queues` | List pending/running/waiting tasks |
| GET | `/api/v1/metrics` | Aggregated workflow + provider metrics |
| GET | `/api/v1/config/compose` | Read docker-compose.yaml |
| PUT | `/api/v1/config/compose` | Write docker-compose.yaml |
| GET | `/api/v1/ws` | WebSocket event stream |
