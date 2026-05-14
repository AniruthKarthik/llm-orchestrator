package observer

import (
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
)

// EventMetricsHandler listens to events and updates metrics.
type EventMetricsHandler struct {
	collector MetricsCollector
}

func NewEventMetricsHandler(collector MetricsCollector) *EventMetricsHandler {
	return &EventMetricsHandler{collector: collector}
}

func (h *EventMetricsHandler) Handle(event events.Event) {
	labels := map[string]string{
		"workflow_id": event.WorkflowID,
		"task_id":     event.TaskID,
		"type":        string(event.Type),
	}

	h.collector.Inc("events_total", labels)

	switch event.Type {
	case events.TaskCompleted:
		if duration, ok := event.Payload["duration"].(time.Duration); ok {
			h.collector.Observe("task_duration_seconds", duration.Seconds(), labels)
		}
	case events.WorkflowCompleted:
		h.collector.Inc("workflows_completed_total", labels)
	case events.WorkflowFailed:
		h.collector.Inc("workflows_failed_total", labels)
	}
}

// RegisterWithBus attaches the handler to the event bus.
func (h *EventMetricsHandler) RegisterWithBus(bus *events.EventBus) {
	bus.Subscribe(events.TaskCompleted, h.Handle)
	bus.Subscribe(events.TaskFailed, h.Handle)
	bus.Subscribe(events.WorkflowCompleted, h.Handle)
	bus.Subscribe(events.WorkflowFailed, h.Handle)
}
