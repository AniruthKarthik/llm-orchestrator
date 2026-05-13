package observer

import (
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
)

type EventLogger struct{}

func NewEventLogger() *EventLogger {
	return &EventLogger{}
}

func (l *EventLogger) Handle(
	event events.Event,
) {
	fmt.Printf(
		"[%s] workflow=%s task=%s payload=%v\n",
		event.Type,
		event.WorkflowID,
		event.TaskID,
		event.Payload,
	)
}
