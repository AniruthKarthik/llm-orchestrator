package memory

import (
	"context"
	"testing"
)

func TestMockEmbedder(t *testing.T) {
	e := NewMockEmbedder()
	ctx := context.Background()

	t.Run("Dimensions", func(t *testing.T) {
		if e.Dimensions() != mockDimensions {
			t.Errorf("expected %d dimensions, got %d", mockDimensions, e.Dimensions())
		}
	})

	t.Run("Embed non-empty string", func(t *testing.T) {
		vec, err := e.Embed(ctx, "hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(vec) != mockDimensions {
			t.Errorf("expected vector length %d, got %d", mockDimensions, len(vec))
		}
	})

	t.Run("Embed empty string returns error", func(t *testing.T) {
		_, err := e.Embed(ctx, "")
		if err == nil {
			t.Error("expected error for empty string, got nil")
		}
	})
}
