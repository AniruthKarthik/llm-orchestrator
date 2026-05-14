package executor

import (
	"container/heap"
	"sync"
)

// QueuedTask represents a task waiting for execution with a priority.
type QueuedTask struct {
	TaskID     string
	WorkflowID string
	Priority   int
	Index      int // The index of the item in the heap.
}

// PriorityQueue implements heap.Interface and holds QueuedTasks.
type PriorityQueue []*QueuedTask

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest priority so we use greater than here.
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*QueuedTask)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// TaskQueue defines the interface for a task queue.
type TaskQueue interface {
	Push(task *QueuedTask) error
	Pop() (*QueuedTask, error)
	Size() int
}

// MemoryTaskQueue is a thread-safe in-memory priority queue.
type MemoryTaskQueue struct {
	pq    PriorityQueue
	mutex sync.Mutex
	cond  *sync.Cond
}

func NewMemoryTaskQueue() *MemoryTaskQueue {
	mq := &MemoryTaskQueue{
		pq: make(PriorityQueue, 0),
	}
	mq.cond = sync.NewCond(&mq.mutex)
	heap.Init(&mq.pq)
	return mq
}

func (q *MemoryTaskQueue) Push(task *QueuedTask) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	heap.Push(&q.pq, task)
	q.cond.Signal()
	return nil
}

func (q *MemoryTaskQueue) Pop() (*QueuedTask, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	for q.pq.Len() == 0 {
		q.cond.Wait()
	}
	item := heap.Pop(&q.pq).(*QueuedTask)
	return item, nil
}

func (q *MemoryTaskQueue) Size() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.pq.Len()
}
