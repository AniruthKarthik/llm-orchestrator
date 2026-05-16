package workers

import "context"

type Worker interface {
	Name() string

	Execute(
		ctx context.Context,
		input map[string]any,
	) (map[string]any, error)
}
