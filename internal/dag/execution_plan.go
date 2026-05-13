package dag

type ExecutionStage struct {
	Level   int
	TaskIDs []string
}

type ExecutionPlan struct {
	WorkflowID string
	Stages     []ExecutionStage
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
