package core

import (
	"encoding/json"
	"fmt"
	"time"
)

// CheckpointData represents the serialized state of a workflow and its tasks.
type CheckpointData struct {
	Workflow *WorkflowSnapshot        `json:"workflow"`
	Tasks    map[string]*TaskSnapshot `json:"tasks"`
}

// WorkflowSnapshot represents the essential state of a workflow.
type WorkflowSnapshot struct {
	ID         string         `json:"id"`
	Status     WorkflowStatus `json:"status"`
	StartedAt  *time.Time     `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at"`
}

// TaskSnapshot represents the essential state of a task.
type TaskSnapshot struct {
	ID         string         `json:"id"`
	Status     TaskStatus     `json:"status"`
	Output     map[string]any `json:"output"`
	Error      string         `json:"error"`
	StartedAt  *time.Time     `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at"`
	Attempt    int            `json:"attempt"`
}

// CreateCheckpoint generates a snapshot of the current workflow state.
func (w *Workflow) CreateCheckpoint() ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	data := CheckpointData{
		Workflow: &WorkflowSnapshot{
			ID:         w.ID,
			Status:     w.Status,
			StartedAt:  w.StartedAt,
			FinishedAt: w.FinishedAt,
		},
		Tasks: make(map[string]*TaskSnapshot),
	}

	for id, task := range w.Tasks {
		task.mu.RLock()
		data.Tasks[id] = &TaskSnapshot{
			ID:         task.ID,
			Status:     task.Status,
			Output:     DeepCopyMap(task.Output),
			Error:      task.Error,
			StartedAt:  task.StartedAt,
			FinishedAt: task.FinishedAt,
			Attempt:    task.Attempt,
		}
		task.mu.RUnlock()
	}

	return json.Marshal(data)
}

// RestoreFromCheckpoint updates the workflow and its tasks from a checkpoint snapshot.
// Tasks that were RUNNING during the checkpoint are reverted to PENDING to allow retry upon resume.
func (w *Workflow) RestoreFromCheckpoint(stateData []byte) error {
	var data CheckpointData
	if err := json.Unmarshal(stateData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint data: %w", err)
	}

	if data.Workflow == nil || data.Workflow.ID != w.ID {
		return fmt.Errorf("invalid checkpoint data for workflow")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.Status = data.Workflow.Status
	// If workflow was running and crashed, keep it running or pending? Wait, the executor will re-start it.
	// Actually, if it's failed, it might stay failed unless we want to resume.
	w.StartedAt = data.Workflow.StartedAt
	w.FinishedAt = data.Workflow.FinishedAt

	for id, taskSnap := range data.Tasks {
		if task, exists := w.Tasks[id]; exists {
			task.mu.Lock()

			status := taskSnap.Status
			// Crash recovery: revert RUNNING tasks to PENDING
			if status == TaskRunning {
				status = TaskPending
				task.StartedAt = nil
			} else {
				task.StartedAt = taskSnap.StartedAt
			}

			task.Status = status
			task.Output = taskSnap.Output
			task.Error = taskSnap.Error
			task.FinishedAt = taskSnap.FinishedAt
			task.Attempt = taskSnap.Attempt

			task.mu.Unlock()
		}
	}

	return nil
}
