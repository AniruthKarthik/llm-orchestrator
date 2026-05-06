package tools

import "context"

type Tool interface {
	Name() string
	Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}
