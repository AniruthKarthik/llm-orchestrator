# LLM Orchestrator Architecture

## Overview
The LLM Orchestrator is a modular, production-grade workflow engine designed for AI agents and LLM-based systems. It supports DAG execution, distributed coordination, resilient retries, and resumable execution.

## Core Components
- **Core Engine:** Manages workflows and tasks with thread-safety and status tracking.
- **DAG Engine:** Handles dependency validation and topological planning for execution stages.
- **Executor:** Orchestrates task execution using worker registries, middleware, and interceptors.
- **Distributed Foundations:** Provides node tracking, heartbeats, and task leasing for multi-node setups.
- **Persistence:** Abstracted storage layer with in-memory implementation for workflows, tasks, and checkpoints.
- **Observability:** Event-driven metrics collection and tracing spans.
- **Plugin & Middleware:** Extensible architecture for custom workers and cross-cutting concerns.
- **LLM Runtime:** Specialized workers for LLM providers with normalized messaging.

## Directory Structure
- `core/`: Task and Workflow models, Retry policies, Checkpointing.
- `dag/`: DAG validation and topological sort.
- `executor/`: Main execution logic, scheduler, queue, limiter, middleware.
- `distributed/`: Coordination, heartbeats, leasing.
- `cluster/`: Node tracking.
- `observer/`: Metrics, tracing, event handlers.
- `plugin/`: Plugin registry and interfaces.
- `llm/`: LLM provider integration and workers.
- `api/`: REST interface.
- `dsl/`: Declarative workflow parser and compiler.
