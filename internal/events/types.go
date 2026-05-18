package events

type EventType string

const (
	WorkflowStarted   EventType = "WORKFLOW_STARTED"
	WorkflowCompleted EventType = "WORKFLOW_COMPLETED"
	WorkflowFailed    EventType = "WORKFLOW_FAILED"

	TaskStarted   EventType = "TASK_STARTED"
	TaskRetried   EventType = "TASK_RETRIED"
	TaskCompleted EventType = "TASK_COMPLETED"
	TaskFailed    EventType = "TASK_FAILED"
	TaskTokenUsage EventType = "TASK_TOKEN_USAGE"

	StageStarted   EventType = "STAGE_STARTED"
	StageCompleted EventType = "STAGE_COMPLETED"
	StageFailed    EventType = "STAGE_FAILED"
)
