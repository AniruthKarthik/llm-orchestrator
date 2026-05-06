package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Span represents one unit of traced work.
type Span interface {
	End(ctx context.Context)
	SetError(err error)
	TraceID() string
	SpanID() string
}

// ── In-process tracer ─────────────────────────────────────────────────────────

// InProcessTracer is a lightweight, zero-dependency tracer that stores spans in memory.  It propagates IDs through context and logs span boundaries via the injected Logger.
type InProcessTracer struct {
	log Logger
}

func NewInProcessTracer(log Logger) *InProcessTracer {
	return &InProcessTracer{log: log}
}

func (t *InProcessTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	traceID := TraceIDFrom(ctx)
	if traceID == "" {
		traceID = newHexID(16)
	}
	spanID := newHexID(8)

	span := &inProcessSpan{
		name:    name,
		traceID: traceID,
		spanID:  spanID,
		log:     t.log,
	}

	// Propagate the trace ID so nested spans inherit it.
	ctx = WithTraceID(ctx, traceID)

	t.log.Debug(ctx, "span started", F("span", name), F("span_id", spanID))
	return ctx, span
}

// concrete Span produced by InProcessTracer.
type inProcessSpan struct {
	name    string
	traceID string
	spanID  string
	errMsg  string
	log     Logger
}

func (s *inProcessSpan) End(ctx context.Context) {
	fields := []Field{
		F("span", s.name),
		F("span_id", s.spanID),
	}
	if s.errMsg != "" {
		fields = append(fields, F("error", s.errMsg))
		s.log.Error(ctx, "span ended with error", fields...)
	} else {
		s.log.Debug(ctx, "span ended", fields...)
	}
}

func (s *inProcessSpan) SetError(err error) {
	if err != nil {
		s.errMsg = err.Error()
	}
}

func (s *inProcessSpan) TraceID() string { return s.traceID }
func (s *inProcessSpan) SpanID() string  { return s.spanID }

// NoopTracer produces no-op spans.  Safe to use when tracing is disabled.
type NoopTracer struct{}

func (NoopTracer) Start(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End(_ context.Context) {}
func (noopSpan) SetError(_ error)      {}
func (noopSpan) TraceID() string       { return "" }
func (noopSpan) SpanID() string        { return "" }

func newHexID(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", byteLen)
	}
	return hex.EncodeToString(b)
}
