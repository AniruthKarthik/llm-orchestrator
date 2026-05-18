package observer

import (
	"fmt"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
	"github.com/AniruthKarthik/llm-orchestrator/internal/store"
)

// AuditLogger records all system events for security and debugging.
type AuditLogger struct {
	store store.Store
}

func NewAuditLogger(s store.Store) *AuditLogger {
	return &AuditLogger{
		store: s,
	}
}

func (l *AuditLogger) Handle(event events.Event) {
	// In a real implementation, we would save this to a specialized audit table
	// For now, we'll just log it to stdout with an "AUDIT" prefix
	// and potentially save it to the database if we add an AuditRecord type.
	
	fmt.Printf("[AUDIT] %s | Workflow: %s | Task: %s | Type: %s | Payload: %v\n",
		event.Timestamp.Format(time.RFC3339),
		event.WorkflowID,
		event.TaskID,
		event.Type,
		event.Payload,
	)
}
