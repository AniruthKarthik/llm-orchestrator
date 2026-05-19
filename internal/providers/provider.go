package providers

import (
	"context"
)

type Provider interface {
	Generate(
		ctx context.Context,
		request GenerateRequest,
	) (*GenerateResponse, error)

	Capabilities() Capabilities

	Name() string

	ListModels(ctx context.Context) ([]string, error)
}
