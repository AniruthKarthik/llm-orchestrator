package observability

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ── Metrics interface ─────────────────────────────────────────────────────────

// Metrics is the hook interface components call to record telemetry.
// All methods are safe to call from multiple goroutines.
// The interface boundary lets callers be instrumented without depending on
// a specific metrics backend (Prometheus, StatsD, …).
type Metrics interface {
	RecordTaskDuration(ctx context.Context, taskType string, d time.Duration, success bool)
	IncTaskRetry(ctx context.Context, taskType string)
	SetQueueSize(ctx context.Context, size int)
	IncWorkerActive(ctx context.Context)
	DecWorkerActive(ctx context.Context)
	Snapshot() MetricsSnapshot
}

// MetricsSnapshot is a read-only copy of the metrics state.
type MetricsSnapshot struct {
	TaskDurations      map[string][]float64 `json:"task_durations_ms"`
	TaskRetryCounts    map[string]int64     `json:"task_retry_counts"`
	QueueSize          int64                `json:"queue_size"`
	ActiveWorkers      int64                `json:"active_workers"`
	TotalTasksExecuted int64                `json:"total_tasks_executed"`
}

// ── In-memory implementation ──────────────────────────────────────────────────

// InMemoryMetrics implements Metrics using only in-process state.
// It is designed to be a drop-in replacement while a real metrics backend is
// wired up; production code should call Snapshot() to feed a metrics exporter.
type InMemoryMetrics struct {
	mu            sync.RWMutex
	durations     map[string][]float64
	retryCounts   map[string]int64
	queueSize     atomic.Int64
	activeWorkers atomic.Int64
	totalExecuted atomic.Int64
}

// NewInMemoryMetrics creates a new instance of InMemoryMetrics.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		durations:   make(map[string][]float64),
		retryCounts: make(map[string]int64),
	}
}

// RecordTaskDuration tracks how long a task took to execute and its outcome.
func (m *InMemoryMetrics) RecordTaskDuration(_ context.Context, taskType string, d time.Duration, success bool) {
	suffix := "success"
	if !success {
		suffix = "failure"
	}
	key := taskType + "/" + suffix
	ms := float64(d.Milliseconds())

	m.mu.Lock()
	m.durations[key] = append(m.durations[key], ms)
	m.mu.Unlock()

	m.totalExecuted.Add(1)
}

// IncTaskRetry increments the retry counter for a specific task type.
func (m *InMemoryMetrics) IncTaskRetry(_ context.Context, taskType string) {
	m.mu.Lock()
	m.retryCounts[taskType]++
	m.mu.Unlock()
}

// SetQueueSize updates the current depth of the task queue.
func (m *InMemoryMetrics) SetQueueSize(_ context.Context, size int) {
	m.queueSize.Store(int64(size))
}

// IncWorkerActive increments the count of workers currently processing tasks.
func (m *InMemoryMetrics) IncWorkerActive(_ context.Context) {
	m.activeWorkers.Add(1)
}

// DecWorkerActive decrements the count of active workers.
func (m *InMemoryMetrics) DecWorkerActive(_ context.Context) {
	m.activeWorkers.Add(-1)
}

// Snapshot returns a point-in-time copy of all recorded metrics.
func (m *InMemoryMetrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	durCopy := make(map[string][]float64, len(m.durations))
	for k, v := range m.durations {
		cp := make([]float64, len(v))
		copy(cp, v)
		durCopy[k] = cp
	}

	retryCopy := make(map[string]int64, len(m.retryCounts))
	for k, v := range m.retryCounts {
		retryCopy[k] = v
	}

	return MetricsSnapshot{
		TaskDurations:      durCopy,
		TaskRetryCounts:    retryCopy,
		QueueSize:          m.queueSize.Load(),
		ActiveWorkers:      m.activeWorkers.Load(),
		TotalTasksExecuted: m.totalExecuted.Load(),
	}
}

// NoopMetrics implements Metrics and discards all data.  Safe for tests.
type NoopMetrics struct{}

// RecordTaskDuration is a no-op.
func (NoopMetrics) RecordTaskDuration(_ context.Context, _ string, _ time.Duration, _ bool) {}

// IncTaskRetry is a no-op.
func (NoopMetrics) IncTaskRetry(_ context.Context, _ string) {}

// SetQueueSize is a no-op.
func (NoopMetrics) SetQueueSize(_ context.Context, _ int) {}

// IncWorkerActive is a no-op.
func (NoopMetrics) IncWorkerActive(_ context.Context) {}

// DecWorkerActive is a no-op.
func (NoopMetrics) DecWorkerActive(_ context.Context) {}

// Snapshot returns an empty MetricsSnapshot.
func (NoopMetrics) Snapshot() MetricsSnapshot { return MetricsSnapshot{} }

// ── convenience bundle ────────────────────────────────────────────────────────

// Obs bundles a Logger, Tracer, and Metrics so components receive a single dependency instead of three.
type Obs struct {
	Log     Logger
	Tracer  Tracer
	Metrics Metrics
}

// NewObs creates a new observability bundle.
func NewObs(l Logger, t Tracer, m Metrics) *Obs {
	return &Obs{Log: l, Tracer: t, Metrics: m}
}

// Default returns a standard observability bundle using JSON logging to stderr.
func Default() *Obs {
	l := NewDefaultLogger()
	return &Obs{
		Log:     l,
		Tracer:  NewInProcessTracer(l),
		Metrics: NewInMemoryMetrics(),
	}
}
