package observer

import (
	"log/slog"
	"time"

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
	record := eventToRecord(event)
	if err := l.store.SaveEvent(record); err != nil {
		slog.Error("audit persist failed",
			"event_type", string(event.Type),
			"workflow_id", event.WorkflowID,
			"task_id", event.TaskID,
			"error", err,
		)
	}

	slog.Info("audit",
		"event_type", string(event.Type),
		"workflow_id", event.WorkflowID,
		"task_id", event.TaskID,
		"severity", record.Severity,
		"message", record.Message,
		"timestamp", event.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"payload", event.Payload,
	)
}

func eventToRecord(event events.Event) store.EventRecord {
	severity := "info"
	switch event.Type {
	case events.WorkflowFailed, events.TaskFailed, events.StageFailed:
		severity = "error"
	case events.TaskRetried, events.TaskWaitingForApproval:
		severity = "warn"
	}

	message := string(event.Type)
	if errMsg, ok := event.Payload["error"].(string); ok && errMsg != "" {
		message = errMsg
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	id := event.ID
	if id == "" {
		id = event.Timestamp.Format("20060102150405.000000000") + "-" + string(event.Type) + "-" + event.WorkflowID + "-" + event.TaskID
	}

	return store.EventRecord{
		ID:         id,
		Type:       string(event.Type),
		WorkflowID: event.WorkflowID,
		TaskID:     event.TaskID,
		Severity:   severity,
		Message:    message,
		Payload:    event.Payload,
		Timestamp:  event.Timestamp,
	}
}
