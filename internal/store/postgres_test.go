package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/core"
)

func TestPostgresStore(t *testing.T) {
	connString := os.Getenv("TEST_DB_URL")
	if connString == "" {
		// Try default local dev string, but skip if it fails immediately
		connString = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check if DB is reachable before proceeding
	s, err := NewPostgresStore(ctx, connString, "../../migrations")
	if err != nil {
		t.Skipf("Skipping PostgresStore test: database not available: %v", err)
		return
	}
	defer s.Close()

	// 1. Test Workflow
	wID := "wf-1"
	workflow := WorkflowRecord{
		ID:          wID,
		Name:        "Test Workflow",
		Description: "A test workflow",
		Status:      string(core.WorkflowPending),
		CreatedAt:   time.Now().Round(time.Microsecond).UTC(), // Postgres precision
		Timeout:     time.Hour,
	}

	err = s.SaveWorkflow(workflow)
	if err != nil {
		t.Errorf("SaveWorkflow failed: %v", err)
	}

	retrievedW, err := s.GetWorkflow(wID)
	if err != nil {
		t.Errorf("GetWorkflow failed: %v", err)
	}
	if retrievedW.Name != workflow.Name {
		t.Errorf("Expected name %s, got %s", workflow.Name, retrievedW.Name)
	}

	workflow.Status = string(core.WorkflowRunning)
	now := time.Now().Round(time.Microsecond).UTC()
	workflow.StartedAt = &now
	err = s.UpdateWorkflow(workflow)
	if err != nil {
		t.Errorf("UpdateWorkflow failed: %v", err)
	}

	retrievedW, _ = s.GetWorkflow(wID)
	if retrievedW.Status != string(core.WorkflowRunning) {
		t.Errorf("Expected status %s, got %s", core.WorkflowRunning, retrievedW.Status)
	}

	// 2. Test Tasks
	tID := "task-1"
	task := TaskRecord{
		ID:           tID,
		WorkflowID:   wID,
		Name:         "Test Task",
		Status:       string(core.TaskPending),
		Input:        map[string]any{"key": "value"},
		Dependencies: []string{},
		CreatedAt:    time.Now().Round(time.Microsecond).UTC(),
		Timeout:      time.Minute,
	}

	err = s.SaveTask(task)
	if err != nil {
		t.Errorf("SaveTask failed: %v", err)
	}

	retrievedT, err := s.GetTask(wID, tID)
	if err != nil {
		t.Errorf("GetTask failed: %v", err)
	}
	if retrievedT.Input["key"] != "value" {
		t.Errorf("Expected input value 'value', got %v", retrievedT.Input["key"])
	}

	task.Status = string(core.TaskCompleted)
	task.Output = map[string]any{"result": 42.0} // JSON numbers are float64
	err = s.UpdateTask(task)
	if err != nil {
		t.Errorf("UpdateTask failed: %v", err)
	}

	tasks, err := s.GetWorkflowTasks(wID)
	if err != nil {
		t.Errorf("GetWorkflowTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
	// Check output
	if tasks[0].Output["result"] != 42.0 {
		t.Errorf("Expected output result 42, got %v", tasks[0].Output["result"])
	}

	// 3. Test Checkpoints
	checkpoint := CheckpointRecord{
		WorkflowID: wID,
		StateData:  []byte(`{"state": "saved"}`),
		Timestamp:  time.Now().Round(time.Microsecond).UTC(),
	}

	err = s.SaveCheckpoint(checkpoint)
	if err != nil {
		t.Errorf("SaveCheckpoint failed: %v", err)
	}

	latestCP, err := s.GetLatestCheckpoint(wID)
	if err != nil {
		t.Errorf("GetLatestCheckpoint failed: %v", err)
	}
	if string(latestCP.StateData) != string(checkpoint.StateData) {
		t.Errorf("Expected state data %s, got %s", checkpoint.StateData, latestCP.StateData)
	}
}
