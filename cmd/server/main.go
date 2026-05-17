package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/AniruthKarthik/llm-orchestrator/internal/api"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/executor"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

type DummyWorker struct{}

func (w *DummyWorker) Execute(ctx context.Context, execCtx *executor.ExecutionContext, task *core.Task) (map[string]any, error) {
	log.Printf("Executing task: %s (%s)", task.ID, task.Name)
	return map[string]any{"status": "success", "taskID": task.ID}, nil
}

func main() {
	// Initialize components
	ms := store.NewMemoryStore()
	eb := events.NewEventBus(10) // 10 workers for event dispatching
	wr := executor.NewWorkerRegistry()

	// Register a dummy worker for testing
	wr.Register("test-task", &DummyWorker{})

	exec := executor.NewExecutor(wr, eb, ms)
	srv := api.NewServer(exec, ms)

	// Setup routes
	handler := srv.Routes()

	port := ":8080"
	fmt.Printf("Starting server on %s\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatal(err)
	}
}
