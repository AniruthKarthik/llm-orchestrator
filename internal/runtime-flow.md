# Runtime Execution Flow

1. **Submission:** A workflow is submitted via API or DSL compiler.
2. **Validation:** The DAG is validated for circular dependencies and missing tasks.
3. **Planning:** The topological planner builds execution stages.
4. **Execution:**
   - For each stage, tasks are executed in parallel.
   - Middlewares (Recover, Auth, Logging) are applied to each task.
   - Retry logic wraps task execution for resilience.
   - Checkpoints are saved after each successful stage.
5. **Completion:** Workflow status is updated to COMPLETED or FAILED based on task results and failure policies.
6. **Observability:** Events are emitted throughout the lifecycle, updating metrics and tracing.
