package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
)

type Store interface {
	SaveJob(job *models.Job) error
	GetJob(id string) (*models.Job, error)
	UpdateJobStatus(id string, status models.JobStatus) error
	UpdateJobTasks(id string, tasks []models.Task) error
	SetJobResult(id string, result any) error
	SetJobError(id string, errMsg string) error
	GetTask(jobID, taskID string) (*models.Task, error)
	UpdateTaskStatus(jobID, taskID string, status models.TaskStatus) error
	SetTaskResult(jobID, taskID string, result any) error
	IncrTaskRetries(jobID, taskID string) error
	AllTasksDone(jobID string) (bool, error)
	AnyTaskFailed(jobID string) (bool, error)
	DepsCompleted(jobID, taskID string) (bool, error)
}

type Memory struct {
	mu   sync.RWMutex
	jobs map[string]*models.Job
}

// New creates a new in-memory data store for jobs and tasks.
func New() *Memory {
	return &Memory{jobs: make(map[string]*models.Job)}
}

// SaveJob persists or updates a job and its associated tasks in the store.
func (m *Memory) SaveJob(job *models.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

// GetJob retrieves a job from the store by its ID, returning a copy.
func (m *Memory) GetJob(id string) (*models.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("store: job %q not found", id)
	}
	cp := *j
	return &cp, nil
}

// UpdateJobStatus modifies the current status of a job in the store.
func (m *Memory) UpdateJobStatus(id string, status models.JobStatus) error {
	return m.mutateJob(id, func(j *models.Job) {
		j.Status = status
		j.UpdatedAt = time.Now().UTC()
	})
}

// UpdateJobTasks overwrites the task list for a specific job.
func (m *Memory) UpdateJobTasks(id string, tasks []models.Task) error {
	return m.mutateJob(id, func(j *models.Job) {
		j.Tasks = tasks
		j.UpdatedAt = time.Now().UTC()
	})
}

// SetJobResult stores the final aggregated result of a completed job.
func (m *Memory) SetJobResult(id string, result any) error {
	return m.mutateJob(id, func(j *models.Job) {
		j.Result = result
		j.Status = models.JobStatusCompleted
		j.UpdatedAt = time.Now().UTC()
	})
}

// SetJobError records a failure message and marks the job as failed.
func (m *Memory) SetJobError(id string, errMsg string) error {
	return m.mutateJob(id, func(j *models.Job) {
		j.Error = errMsg
		j.Status = models.JobStatusFailed
		j.UpdatedAt = time.Now().UTC()
	})
}

// GetTask finds and returns a copy of a specific task within a job.
func (m *Memory) GetTask(jobID, taskID string) (*models.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, err := m.findTask(jobID, taskID)
	if err != nil {
		return nil, err
	}
	cp := *t
	return &cp, nil
}

// UpdateTaskStatus changes the current status of a task.
func (m *Memory) UpdateTaskStatus(jobID, taskID string, status models.TaskStatus) error {
	return m.mutateTask(jobID, taskID, func(t *models.Task) {
		t.Status = status
	})
}

// SetTaskResult stores the outcome of a successfully executed task.
func (m *Memory) SetTaskResult(jobID, taskID string, result any) error {
	return m.mutateTask(jobID, taskID, func(t *models.Task) {
		t.Result = result
		t.Status = models.TaskStatusCompleted
	})
}

// IncrTaskRetries increments the retry counter for a task.
func (m *Memory) IncrTaskRetries(jobID, taskID string) error {
	return m.mutateTask(jobID, taskID, func(t *models.Task) {
		t.Retries++
	})
}

// AllTasksDone checks if every task in a job has finished (success or failure).
func (m *Memory) AllTasksDone(jobID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return false, fmt.Errorf("store: job %q not found", jobID)
	}
	for i := range j.Tasks {
		s := j.Tasks[i].Status
		if s != models.TaskStatusCompleted && s != models.TaskStatusFailed {
			return false, nil
		}
	}
	return true, nil
}

// AnyTaskFailed determines if at least one task in the job has failed.
func (m *Memory) AnyTaskFailed(jobID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return false, fmt.Errorf("store: job %q not found", jobID)
	}
	for i := range j.Tasks {
		if j.Tasks[i].Status == models.TaskStatusFailed {
			return true, nil
		}
	}
	return false, nil
}

// DepsCompleted checks if all dependencies of a specific task are completed.
func (m *Memory) DepsCompleted(jobID, taskID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	j, ok := m.jobs[jobID]
	if !ok {
		return false, fmt.Errorf("store: job %q not found", jobID)
	}
	done := make(map[string]bool, len(j.Tasks))
	for i := range j.Tasks {
		if j.Tasks[i].Status == models.TaskStatusCompleted {
			done[j.Tasks[i].ID] = true
		}
	}
	for i := range j.Tasks {
		if j.Tasks[i].ID == taskID {
			for _, dep := range j.Tasks[i].DependsOn {
				if !done[dep] {
					return false, nil
				}
			}
			return true, nil
		}
	}
	return false, fmt.Errorf("store: task %q not found in job %q", taskID, jobID)
}

// mutateJob applies a mutation function to a job within a write lock.
func (m *Memory) mutateJob(id string, fn func(*models.Job)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("store: job %q not found", id)
	}
	fn(j)
	return nil
}

// mutateTask applies a mutation function to a specific task within a write lock.
func (m *Memory) mutateTask(jobID, taskID string, fn func(*models.Task)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, err := m.findTask(jobID, taskID)
	if err != nil {
		return err
	}
	fn(t)
	return nil
}

// findTask retrieves a pointer to a specific task (internal use, requires locking).
func (m *Memory) findTask(jobID, taskID string) (*models.Task, error) {
	j, ok := m.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("store: job %q not found", jobID)
	}
	for i := range j.Tasks {
		if j.Tasks[i].ID == taskID {
			return &j.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("store: task %q not found in job %q", taskID, jobID)
}
