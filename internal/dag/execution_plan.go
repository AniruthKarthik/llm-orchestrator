package dag

type ExecutionStage struct {
	Level   int      `json:"level"`
	TaskIDs []string `json:"taskIds"`
}

type ExecutionPlan struct {
	WorkflowID string           `json:"workflowId"`
	Stages     []ExecutionStage `json:"stages"`
}

func NewExecutionPlan(workflowID string) *ExecutionPlan {
	return &ExecutionPlan{
		WorkflowID: workflowID,
		Stages:     []ExecutionStage{},
	}
}

func (p *ExecutionPlan) AddStage(stage ExecutionStage) {
	p.Stages = append(p.Stages, stage)
}
