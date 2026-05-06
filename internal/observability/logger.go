package observability

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"
)

type ctxKey string

const (
	traceIDKey  ctxKey = "trace_id"
	jobIDKey    ctxKey = "job_id"
	taskIDKey   ctxKey = "task_id"
	workerIDKey ctxKey = "worker_id"
)

// WithTraceID attaches a trace ID to the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// TraceIDFrom extracts the trace ID from ctx, returning "" if absent.
func TraceIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// WithJobID attaches a job ID to ctx.
func WithJobID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, jobIDKey, id)
}

// JobIDFrom extracts the job ID from ctx.
func JobIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(jobIDKey).(string)
	return v
}

// WithTaskID attaches a task ID to ctx.
func WithTaskID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, taskIDKey, id)
}

// TaskIDFrom extracts the task ID from ctx.
func TaskIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(taskIDKey).(string)
	return v
}

// WithWorkerID attaches a worker ID to ctx.
func WithWorkerID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, workerIDKey, id)
}

// WorkerIDFrom extracts the worker ID from ctx.
func WorkerIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(workerIDKey).(string)
	return v
}

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
}

type Field struct {
	Key   string
	Value any
}

// F creates a new logging field.
func F(key string, value any) Field { return Field{Key: key, Value: value} }

type entry struct {
	Timestamp string         `json:"ts"`
	Level     Level          `json:"level"`
	Message   string         `json:"msg"`
	TraceID   string         `json:"trace_id,omitempty"`
	JobID     string         `json:"job_id,omitempty"`
	TaskID    string         `json:"task_id,omitempty"`
	WorkerID  string         `json:"worker_id,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// JSONLogger writes one JSON object per log call to w.
type JSONLogger struct {
	w      io.Writer
	minLvl Level
	enc    *json.Encoder
}

// NewJSONLogger creates a JSONLogger that writes to w.
func NewJSONLogger(w io.Writer, minLevel Level) *JSONLogger {
	if w == nil {
		w = os.Stderr
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &JSONLogger{w: w, minLvl: minLevel, enc: enc}
}

// NewDefaultLogger returns a JSONLogger writing to stderr at INFO level.
func NewDefaultLogger() *JSONLogger {
	return NewJSONLogger(os.Stderr, LevelInfo)
}

// log handles the internal logic of formatting and writing a log entry.
func (l *JSONLogger) log(ctx context.Context, lvl Level, msg string, fields []Field) {
	if !l.shouldLog(lvl) {
		return
	}
	e := entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     lvl,
		Message:   msg,
		TraceID:   TraceIDFrom(ctx),
		JobID:     JobIDFrom(ctx),
		TaskID:    TaskIDFrom(ctx),
		WorkerID:  WorkerIDFrom(ctx),
	}
	if len(fields) > 0 {
		e.Fields = make(map[string]any, len(fields))
		for _, f := range fields {
			e.Fields[f.Key] = f.Value
		}
	}
	// Encoder is not goroutine-safe — use a single encode call with a
	// pre-built value so the lock window is as small as possible.
	// For higher throughput, callers can provide a buffered writer.
	_ = l.enc.Encode(e)
}

// shouldLog returns true if the given level is greater than or equal to the logger's minimum level.
func (l *JSONLogger) shouldLog(lvl Level) bool {
	order := map[Level]int{LevelDebug: 0, LevelInfo: 1, LevelWarn: 2, LevelError: 3}
	return order[lvl] >= order[l.minLvl]
}

// Debug logs a message at the DEBUG level.
func (l *JSONLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelDebug, msg, fields)
}

// Info logs a message at the INFO level.
func (l *JSONLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelInfo, msg, fields)
}

// Warn logs a message at the WARN level.
func (l *JSONLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelWarn, msg, fields)
}

// Error logs a message at the ERROR level.
func (l *JSONLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, LevelError, msg, fields)
}

// NoopLogger discards all log entries.  Useful in tests.
type NoopLogger struct{}

// Debug is a no-op.
func (NoopLogger) Debug(_ context.Context, _ string, _ ...Field) {}

// Info is a no-op.
func (NoopLogger) Info(_ context.Context, _ string, _ ...Field) {}

// Warn is a no-op.
func (NoopLogger) Warn(_ context.Context, _ string, _ ...Field) {}

// Error is a no-op.
func (NoopLogger) Error(_ context.Context, _ string, _ ...Field) {}
