package queue

import (
	"fmt"
	"github.com/AniruthKarthik/llm-orchestrator/internal/models"
	"log"
	"sync"
)

const defaultBufferSize = 1024

type Queue interface {
	Enqueue(task models.Task) error
	Dequeue() (models.Task, bool)
	Requeue(task models.Task) error
	Close()
	Len() int
}

type MemoryQueue struct {
	ch     chan models.Task
	once   sync.Once
	closed bool
	mu     sync.Mutex
}

func New(bufferSize int) *MemoryQueue {
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}
	return &MemoryQueue{ch: make(chan models.Task, bufferSize)}
}

func (q *MemoryQueue) Enqueue(task models.Task) error {
	q.mu.Lock()
	closed := q.closed
	q.mu.Unlock()
	if closed {
		return fmt.Errorf("queue: enqueue on closed queue (task %s)", task.ID)
	}
	q.ch <- task
	log.Printf("[queue] enqueued task %s (job %s, type %s)", task.ID, task.JobID, task.Type)
	return nil
}

func (q *MemoryQueue) Dequeue() (models.Task, bool) {
	task, ok := <-q.ch
	return task, ok
}

func (q *MemoryQueue) Requeue(task models.Task) error {
	q.mu.Lock()
	closed := q.closed
	q.mu.Unlock()
	if closed {
		return fmt.Errorf("queue: requeue on closed queue (task %s)", task.ID)
	}
	select {
	case q.ch <- task:
		log.Printf("[queue] requeued task %s (job %s)", task.ID, task.JobID)
		return nil
	default:
		return fmt.Errorf("queue: buffer full, could not requeue task %s", task.ID)
	}
}

func (q *MemoryQueue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		q.closed = true
		q.mu.Unlock()
		close(q.ch)
		log.Println("[queue] closed")
	})
}

func (q *MemoryQueue) Len() int {
	return len(q.ch)
}
