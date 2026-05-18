# LLM Orchestrator Architecture

## Overview
The LLM Orchestrator is a modular, production-grade workflow engine designed for AI agents and LLM-based systems. It is optimized for local execution on a single machine, providing a robust foundation for building autonomous agents.

## Core Components
- **Core Engine:** Manages workflows and tasks with thread-safety and status tracking.
- **DAG Engine:** Handles dependency validation and topological planning for execution stages.
- **Executor:** Orchestrates task execution using worker registries, middleware, and interceptors.
- **Persistence:** Abstracted storage layer with PostgreSQL and in-memory implementations for workflows, tasks, and checkpoints.
- **Observability:** Event-driven metrics collection and tracing.
- **Plugin & Middleware:** Extensible architecture for custom workers and cross-cutting concerns.
- **LLM Runtime:** Specialized workers for LLM providers with normalized messaging.

## Directory Structure
- `core/`: Task and Workflow models, Retry policies, Checkpointing.
- `dag/`: DAG validation and topological sort.
- `executor/`: Main execution logic, scheduler, queue, limiter, middleware.
- `observer/`: Metrics, tracing, event handlers.
- `plugin/`: Plugin registry and interfaces.
- `llm/`: LLM provider integration and workers.
- `api/`: REST interface.
- `dsl/`: Declarative workflow parser and compiler.
- `store/`: Persistence implementations (Postgres, Memory).
- `providers/`: LLM provider implementations (OpenAI, Anthropic, Gemini, Groq).
