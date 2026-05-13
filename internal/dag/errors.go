package dag

import "errors"

var (
	ErrTaskNotFound       = errors.New("task not found")
	ErrDuplicateTask      = errors.New("duplicate task")
	ErrCircularDependency = errors.New("circular dependency")
	ErrSelfDependency     = errors.New("task cannot depend on itself")
)
