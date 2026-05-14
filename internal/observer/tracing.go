package observer

import (
	"context"
	"time"
)

// Span represents a single unit of work in a trace.
type Span struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]string
}

// TracingProvider defines the interface for creating and managing spans.
type TracingProvider interface {
	StartSpan(ctx context.Context, name string) (context.Context, *Span)
	EndSpan(span *Span)
}

// DefaultTracingProvider is a mock/simple implementation of TracingProvider.
type DefaultTracingProvider struct{}

func (p *DefaultTracingProvider) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	span := &Span{
		Name:      name,
		StartTime: time.Now(),
	}
	return ctx, span
}

func (p *DefaultTracingProvider) EndSpan(span *Span) {
	span.EndTime = time.Now()
}
