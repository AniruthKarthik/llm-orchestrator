package events

type EventType string

const (
	WorkflowStarted   EventType = "WORKFLOW_STARTED"
	WorkflowCompleted EventType = "WORKFLOW_COMPLETED"
	WorkflowFailed    EventType = "WORKFLOW_FAILED"

	TaskStarted   EventType = "TASK_STARTED"
	TaskCompleted EventType = "TASK_COMPLETED"
	TaskFailed    EventType = "TASK_FAILED"

	StageStarted   EventType = "STAGE_STARTED"
	StageCompleted EventType = "STAGE_COMPLETED"
	StageFailed    EventType = "STAGE_FAILED"
)
