package executor

type TaskResult struct {
	TaskID  string
	Success bool
	Output  map[string]any
	Error   error
}

type StageResult struct {
	Level   int
	Success bool
	Results []TaskResult
	Errors  []error
}
