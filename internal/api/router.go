package api

import (
	"log"
	"net/http"
	"time"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /job", h.CreateJob)
	mux.HandleFunc("GET /job/{id}", h.GetJob)
	mux.HandleFunc("GET /job/{id}/result", h.GetJobResult)

	return loggingMiddleware(mux)
}

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

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
