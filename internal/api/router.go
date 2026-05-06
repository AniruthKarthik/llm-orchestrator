package api

import (
	"log"
	"net/http"
	"time"
)

// NewRouter sets up the HTTP routes and applies global logging middleware.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /job", h.CreateJob)
	mux.HandleFunc("GET /job/{id}", h.GetJob)
	mux.HandleFunc("GET /job/{id}/result", h.GetJobResult)

	return loggingMiddleware(mux)
}

// loggingMiddleware wraps a handler to log details about every HTTP request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{
			ResponseWriter: w, status: http.StatusOK,
		}

		next.ServeHTTP(rw, r)
		log.Printf("[http] %s %s -> %d (%s)", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before sending it to the client.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
