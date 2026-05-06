package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
)

// NewRouter constructs the HTTP mux and applies middleware.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /job", h.CreateJob)
	mux.HandleFunc("GET /job/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/job/")
		switch {
		case strings.HasSuffix(rest, "/result"):
			h.GetJobResult(w, r)
		default:
			h.GetJob(w, r)
		}
	})
	mux.HandleFunc("GET /metrics", h.GetMetrics)

	return loggingMiddleware(mux, h.obs)
}

// loggingMiddleware wraps an http.Handler to provide request logging and tracing.
func loggingMiddleware(next http.Handler, obs *observability.Obs) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, span := obs.Tracer.Start(r.Context(), "http.request")
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		elapsed := time.Since(start)
		obs.Log.Info(ctx, "http request",
			observability.F("method", r.Method),
			observability.F("path", r.URL.Path),
			observability.F("status", rw.status),
			observability.F("duration_ms", elapsed.Milliseconds()),
			observability.F("trace_id", span.TraceID()),
		)
		span.End(ctx)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before sending it to the underlying ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
