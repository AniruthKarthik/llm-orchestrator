package executor

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/agents"
	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

func TestConcurrencyStress(t *testing.T) {
	// Setup dependencies
	s := store.NewMemoryStore()
	eb := events.NewEventBus(100)
	wr := NewWorkerRegistry()
	ar := agents.NewAgentRegistry()
	art := core.NewArtifactRegistry()
	mem := core.NewMemoryRegistry()
	tr := core.NewToolRegistry()
	tp := core.NewToolPolicy()

	// Register a simple worker
	wr.Register("test-task", &dummyWorker{})

	exec := NewExecutor(wr, ar, art, mem, tr, tp, eb, s)
	exec.WithConcurrencyLimit(50)

	var wg sync.WaitGroup
	numWorkflows := 20
	errChan := make(chan error, numWorkflows)

	for i := 0; i < numWorkflows; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			wfID := fmt.Sprintf("wf-%d", id)
			workflow := core.NewWorkflow(wfID, "Stress Workflow", "Testing concurrency")
			
			// Add 5 tasks to each workflow
			for j := 0; j < 5; j++ {
				taskID := fmt.Sprintf("t-%d-%d", id, j)
				var deps []string
				if j > 0 {
					deps = []string{fmt.Sprintf("t-%d-%d", id, j-1)}
				}
				task := core.NewTask(taskID, wfID, "test-task", "desc", nil, deps)
				workflow.AddTask(task)
			}

			if err := exec.Execute(workflow); err != nil {
				errChan <- fmt.Errorf("workflow %d failed: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Error(err)
		}
	}
}

type dummyWorker struct{}

func (w *dummyWorker) Execute(ctx context.Context, execCtx *ExecutionContext, task *core.Task) (map[string]any, error) {
	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	return map[string]any{"status": "ok"}, nil
}
