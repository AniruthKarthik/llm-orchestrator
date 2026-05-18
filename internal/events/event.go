package events

import (
	"time"
)

type Event struct {
	ID   string    `json:"id,omitempty"`
	Type EventType `json:"type"`

	WorkflowID string `json:"workflowId"`
	TaskID     string `json:"taskId,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Payload map[string]any `json:"payload,omitempty"`
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
