package observer

import (
	"fmt"

	"github.com/AniruthKarthik/llm-orchestrator/internal/events"
)

// MetricsEventHandler translates system events into metrics.
type MetricsEventHandler struct {
	collector MetricsCollector
}

func NewMetricsEventHandler(c MetricsCollector) *MetricsEventHandler {
	return &MetricsEventHandler{
		collector: c,
	}
}

func (h *MetricsEventHandler) Handle(event events.Event) {
	switch event.Type {
	case events.TaskStarted:
		h.collector.Inc("tasks_started_total", map[string]string{
			"workflow_id": event.WorkflowID,
		})
	case events.TaskCompleted:
		h.collector.Inc("tasks_completed_total", map[string]string{
			"workflow_id": event.WorkflowID,
		})
	case events.TaskFailed:
		h.collector.Inc("tasks_failed_total", map[string]string{
			"workflow_id": event.WorkflowID,
		})
	case events.TaskTokenUsage:
		if promptTokens, ok := event.Payload["prompt_tokens"].(int); ok {
			h.collector.Observe("prompt_tokens_total", float64(promptTokens), map[string]string{
				"workflow_id": event.WorkflowID,
				"model":       fmt.Sprintf("%v", event.Payload["model"]),
			})
		}
		if completionTokens, ok := event.Payload["completion_tokens"].(int); ok {
			h.collector.Observe("completion_tokens_total", float64(completionTokens), map[string]string{
				"workflow_id": event.WorkflowID,
				"model":       fmt.Sprintf("%v", event.Payload["model"]),
			})
		}
	}
}
