package providers

import (
	"context"
)

type Provider interface {
	Generate(
		ctx context.Context,
		request GenerateRequest,
	) (*GenerateResponse, error)

	Stream(
		ctx context.Context,
		request GenerateRequest,
	) (<-chan StreamChunk, <-chan error)

	Capabilities() Capabilities

	Name() string

	ListModels(ctx context.Context) ([]string, error)
}
