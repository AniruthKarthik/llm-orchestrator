package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
)

// DeadLetterEntry is a record of a permanently-failed task.
type DeadLetterEntry struct {
	ID       string    `json:"id"`
	JobID    string    `json:"job_id"`
	TaskID   string    `json:"task_id"`
	TaskType string    `json:"task_type"`
	Reason   string    `json:"reason"`
	Payload  any       `json:"payload"`
	Retries  int       `json:"retries"`
	FailedAt time.Time `json:"failed_at"`
}

// DeadLetterQueue is the interface for permanent-failure storage. In production this would write to the dead_letter_tasks Postgres table.
type DeadLetterQueue interface {
	Enqueue(ctx context.Context, entry DeadLetterEntry) error
	List(ctx context.Context, jobID string) ([]DeadLetterEntry, error)
	Size() int
}

// MemoryDLQ is a thread-safe, in-memory DeadLetterQueue.
type MemoryDLQ struct {
	mu      sync.RWMutex
	entries []DeadLetterEntry
}

func NewMemoryDLQ() *MemoryDLQ {
	return &MemoryDLQ{}
}

func (q *MemoryDLQ) Enqueue(_ context.Context, entry DeadLetterEntry) error {
	if entry.TaskID == "" {
		return fmt.Errorf("dlq: task_id must not be empty")
	}
	entry.FailedAt = time.Now().UTC()

	q.mu.Lock()
	q.entries = append(q.entries, entry)
	q.mu.Unlock()
	return nil
}

func (q *MemoryDLQ) List(_ context.Context, jobID string) ([]DeadLetterEntry, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if jobID == "" {
		cp := make([]DeadLetterEntry, len(q.entries))
		copy(cp, q.entries)
		return cp, nil
	}

	var out []DeadLetterEntry
	for _, e := range q.entries {
		if e.JobID == jobID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (q *MemoryDLQ) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.entries)
}

// SendToDLQ is a convenience function that builds a DeadLetterEntry from a task and sends it to the DLQ.  It is called by the executor when a task exhausts its retries.
func SendToDLQ(ctx context.Context, dlq DeadLetterQueue, task models.Task, reason string) error {
	entry := DeadLetterEntry{
		ID:       fmt.Sprintf("dlq-%s-%s", task.JobID, task.ID),
		JobID:    task.JobID,
		TaskID:   task.ID,
		TaskType: task.Type,
		Reason:   reason,
		Payload:  task.Payload,
		Retries:  task.Retries,
		FailedAt: time.Now().UTC(),
	}
	return dlq.Enqueue(ctx, entry)
}
