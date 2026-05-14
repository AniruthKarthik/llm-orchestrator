# Workflow DSL

## Definition
Workflows can be defined declaratively using JSON (and YAML support via tags).

### Example JSON
```json
{
  "id": "workflow-1",
  "name": "LLM Search",
  "tasks": [
    {
      "id": "task-1",
      "name": "search-worker",
      "input": {"query": "Gemini CLI"}
    },
    {
      "id": "task-2",
      "name": "llm-worker",
      "dependencies": ["task-1"],
      "input": {"model": "gpt-4"}
    }
  ]
}
```

## Compilation
1. `Parser` parses the data into a `WorkflowDefinition`.
2. `Compiler` validates the definition and builds a runtime `Workflow` with tasks and dependencies correctly linked.
