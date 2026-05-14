# Plugin System

## Interfaces
- `Plugin`: Base interface for all extensions.
- `ExecutionInterceptor`: Hooks into the task lifecycle (`BeforeTask`, `AfterTask`).

## Types
- `WORKER`: Custom task execution logic.
- `OBSERVER`: Metrics, logging, and tracing extensions.
- `PERSISTENCE`: Custom storage backends.

## Middleware Pipeline
- `TaskMiddleware`: Wraps individual task handlers for cross-cutting logic.
- `WorkflowMiddleware`: Wraps the entire workflow execution.
- Chains are built using `ApplyTaskMiddleware` and `ApplyWorkflowMiddleware`.
