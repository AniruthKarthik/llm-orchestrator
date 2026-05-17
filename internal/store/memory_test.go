package store

import (
	"testing"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestMemoryStore_DeepCopy(t *testing.T) {
	s := NewMemoryStore()
	workflowID := "w1"
	s.SaveWorkflow(WorkflowRecord{ID: workflowID})

	nestedMap := map[string]any{
		"key1": "val1",
		"nested": map[string]any{
			"inner": "orig",
		},
	}

	task := TaskRecord{
		ID:         "t1",
		WorkflowID: workflowID,
		Input:      nestedMap,
	}

	// Save task
	err := s.SaveTask(task)
	if err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	// Mutate the original map
	nestedMap["nested"].(map[string]any)["inner"] = "mutated"

	// Retrieve task from store
	retrieved, err := s.GetTask(workflowID, "t1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	// Check if retrieved task has original value
	innerVal := retrieved.Input["nested"].(map[string]any)["inner"]
	if innerVal != "orig" {
		t.Errorf("expected inner value 'orig', got '%v'", innerVal)
	}

	// Mutate retrieved task
	retrieved.Input["nested"].(map[string]any)["inner"] = "mutated_again"

	// Retrieve again
	retrieved2, _ := s.GetTask(workflowID, "t1")
	innerVal2 := retrieved2.Input["nested"].(map[string]any)["inner"]
	if innerVal2 != "orig" {
		t.Errorf("expected inner value 'orig' on second retrieval, got '%v'", innerVal2)
	}
}

func TestCoreTask_DeepCopy(t *testing.T) {
	input := map[string]any{
		"nested": map[string]any{
			"key": "val",
		},
	}
	task := core.NewTask("t1", "name", "desc", input, nil)

	// Mutate input
	input["nested"].(map[string]any)["key"] = "changed"

	// Task input should remain "val"
	taskInput := task.Input["nested"].(map[string]any)["key"]
	if taskInput != "val" {
		t.Errorf("expected 'val', got '%v'", taskInput)
	}

	// Mutate task input via GetOutput (actually task.Output is empty, let's use Complete)
	output := map[string]any{
		"res": map[string]any{"ok": true},
	}
	task.Start()
	task.Complete(output)

	// Mutate output map
	output["res"].(map[string]any)["ok"] = false

	// Task output should remain true
	taskOutput := task.GetOutput()["res"].(map[string]any)["ok"]
	if taskOutput != true {
		t.Errorf("expected true, got '%v'", taskOutput)
	}
}
