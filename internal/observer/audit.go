package observer

import (
	"log/slog"

	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

// AuditLogger records all system events as structured log lines.
// In a future iteration this can persist to a dedicated audit table in the store.
type AuditLogger struct {
	store store.Store
}

func NewAuditLogger(s store.Store) *AuditLogger {
	return &AuditLogger{store: s}
}

func (l *AuditLogger) Handle(event events.Event) {
	slog.Info("audit",
		"event_type", string(event.Type),
		"workflow_id", event.WorkflowID,
		"task_id", event.TaskID,
		"timestamp", event.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"payload", event.Payload,
	)
}
