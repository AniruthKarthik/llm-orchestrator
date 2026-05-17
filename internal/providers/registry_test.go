package providers

import (
	"context"
	"testing"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	return nil, nil
}
func (m *mockProvider) Stream(ctx context.Context, req GenerateRequest) (<-chan StreamChunk, <-chan error) {
	return nil, nil
}
func (m *mockProvider) Capabilities() Capabilities {
	return Capabilities{SupportsTools: true}
}

func TestRegistry(t *testing.T) {
	Clear()

	p1 := &mockProvider{name: "p1"}
	p2 := &mockProvider{name: "p2"}

	Register(p1)
	Register(p2)

	// Test List
	list := List()
	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}

	// Test Get
	got, err := Get("p1")
	if err != nil {
		t.Fatalf("failed to get p1: %v", err)
	}
	if got.Name() != "p1" {
		t.Errorf("expected p1, got %s", got.Name())
	}

	if got.Capabilities().SupportsTools != true {
		t.Errorf("expected p1 to support tools")
	}

	// Test Not Found
	_, err = Get("missing")
	if err == nil {
		t.Error("expected error for missing provider, got nil")
	}

	// Test Clear
	Clear()
	if len(List()) != 0 {
		t.Error("expected empty registry after clear")
	}
}
