package events

import (
	"time"
)

type Event struct {
	ID   string
	Type EventType

	WorkflowID string
	TaskID     string

	Timestamp time.Time

	Payload map[string]any
}

func NewEvent(
	eventType EventType,
	workflowID string,
	taskID string,
	payload map[string]any,
) Event {
	payloadCopy := make(map[string]any)
	if payload != nil {
		for k, v := range payload {
			payloadCopy[k] = v
		}
	}

	return Event{
		Type:       eventType,
		WorkflowID: workflowID,
		TaskID:     taskID,
		Payload:    payloadCopy,
		Timestamp:  time.Now(),
	}
}
