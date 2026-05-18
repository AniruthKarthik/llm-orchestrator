package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
	"github.com/AniruthKarthik/llm-orchestrator/internal/providers/groq"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
)

// LLMWorker implements executor.Worker using providers.Provider
type LLMWorker struct {
	provider providers.Provider
}

func (w *LLMWorker) Execute(
	ctx context.Context,
	execCtx *executor.ExecutionContext,
	task *core.Task,
) (map[string]any, error) {
	fmt.Printf("[Worker] Executing task %s (%s)\n", task.ID, task.Description)
	
	model, ok := task.Input["model"].(string)
	if !ok {
		return nil, fmt.Errorf("model not found in task input")
	}
	prompt, ok := task.Input["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt not found in task input")
	}

	resp, err := w.provider.Generate(ctx, providers.GenerateRequest{
		Model: model,
		Messages: []providers.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"response": resp.Content,
	}, nil
}

func main() {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GROQ_API_KEY environment variable is required")
		os.Exit(1)
	}

	// 1. Initialize dependencies
	registry := executor.NewWorkerRegistry()
	eventBus := events.NewEventBus(5)
	memoryStore := store.NewMemoryStore()
	agentRegistry := agents.NewAgentRegistry()
	artifactRegistry := core.NewArtifactRegistry()
	memoryRegistry := core.NewMemoryRegistry()
	toolRegistry := core.NewToolRegistry()
	toolPolicy := core.NewToolPolicy()

	// 2. Register worker
	groqProvider := groq.NewGroqProvider(apiKey)
	llmWorker := &LLMWorker{provider: groqProvider}
	registry.Register("llm", llmWorker)

	// Subscribe to events for logging
	eventBus.Subscribe(events.TaskStarted, func(e events.Event) {
		fmt.Printf("[Event] Task %s started\n", e.TaskID)
	})
	eventBus.Subscribe(events.TaskCompleted, func(e events.Event) {
		fmt.Printf("[Event] Task %s completed\n", e.TaskID)
	})
	eventBus.Subscribe(events.TaskFailed, func(e events.Event) {
		fmt.Printf("[Event] Task %s failed: %v\n", e.TaskID, e.Payload["error"])
	})

	// 3. Build workflow
	workflow := core.NewWorkflow("wf-groq-test", "Groq Integration Test", "Verifying Groq provider integration")
	
	taskA := core.NewTask("task-1", "llm", "Explain orchestration systems", map[string]any{
		"model":  "llama3-8b-8192", // Common Groq model
		"prompt": "Explain orchestration systems in 2 sentences.",
	}, nil)
	
	if err := workflow.AddTask(taskA); err != nil {
		fmt.Printf("Failed to add task: %v\n", err)
		os.Exit(1)
	}

	// 4. Execute workflow
	exec := executor.NewExecutor(registry, agentRegistry, artifactRegistry, memoryRegistry, toolRegistry, toolPolicy, eventBus, memoryStore)
	
	fmt.Println("Starting workflow execution...")
	err := exec.Execute(workflow)
	if err != nil {
		fmt.Printf("Workflow execution failed: %v\n", err)
		os.Exit(1)
	}

	// 5. Print result
	completedTask, err := workflow.GetTask("task-1")
	if err != nil {
		fmt.Printf("Failed to get task: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nWorkflow Result:")
	fmt.Printf("Status: %s\n", workflow.Status)
	fmt.Printf("Response: %v\n", completedTask.Output["response"])
}
