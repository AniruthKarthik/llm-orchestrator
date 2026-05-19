# Future Features & Roadmap

This document outlines the planned enhancements for the LLM Orchestrator to evolve it into a more powerful and production-ready platform.

## 1. Advanced Intelligence (The "Autopilot" Path)
- **AI Planner:** Re-implement the `Planner` to automatically generate tasks and dependencies from a high-level goal using LLMs.
- **Semantic Routing:** Implement a router that uses embeddings or a small model to choose the best agent for a task based on description analysis.

## 2. Developer Experience (The "Local-First" Path)
- **Code Export:** Export visual workflows as standalone Go or Python scripts.
- **Local Tool Integration:** Expand the `WorkerRegistry` with built-in tools like a Python Interpreter, web searchers, and file system utilities.
- **Shared Memory:** Implement a "Shared Workspace" (KV store) for cross-task data sharing beyond standard artifacts.

## 3. Production Hardening (The "Enterprise" Path)
- **Workflow Versioning:** Version control for workflow definitions to support rollbacks.
- **RBAC:** User accounts and role-based permissions for the API and UI.
- **Enhanced Resumption:** Improvements to the `Supervisor` for more robust task recovery after system failures.
