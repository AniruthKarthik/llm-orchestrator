package api

import (
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/observability"
	"github.com/AniruthKarthik/llm-orchestrator/internal/utils"
)

// RecoveryMiddleware catches panics in downstream handlers, logs the stack trace, and returns a 500 response so the server keeps running.
func RecoveryMiddleware(next http.Handler, obs *observability.Obs) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				obs.Log.Error(r.Context(), "panic recovered",
					observability.F("panic", rec),
					observability.F("stack", string(stack)),
					observability.F("path", r.URL.Path),
					observability.F("method", r.Method),
				)

				utils.WriteError(w, http.StatusInternalServerError, "internal server error")

			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ipBucket tracks the token count for one IP address.
type ipBucket struct {
	mu       sync.Mutex
	tokens   int
	lastFill time.Time
}

// RateLimiter implements a per-IP token bucket at requestsPerMinute capacity.
type RateLimiter struct {
	mu                sync.RWMutex
	buckets           map[string]*ipBucket
	requestsPerMinute int
	cleanupEvery      time.Duration
	lastCleanup       time.Time
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		buckets:           make(map[string]*ipBucket),
		requestsPerMinute: requestsPerMinute,
		cleanupEvery:      5 * time.Minute,
		lastCleanup:       time.Now(),
	}
}

// Allow returns true if the request from ip should proceed.
func (rl *RateLimiter) Allow(ip string) bool {
	bucket := rl.getBucket(ip)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastFill)

	refill := int(elapsed.Seconds() * float64(rl.requestsPerMinute) / 60.0)
	if refill > 0 {
		bucket.tokens += refill
		if bucket.tokens > rl.requestsPerMinute {
			bucket.tokens = rl.requestsPerMinute
		}
		bucket.lastFill = now
	}

	if bucket.tokens <= 0 {
		return false
	}
	bucket.tokens--
	return true
}

func (rl *RateLimiter) getBucket(ip string) *ipBucket {
	rl.mu.RLock()
	b, ok := rl.buckets[ip]
	rl.mu.RUnlock()
	if ok {
		return b
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if b, ok = rl.buckets[ip]; ok {
		return b
	}

	b = &ipBucket{
		tokens:   rl.requestsPerMinute,
		lastFill: time.Now(),
	}
	rl.buckets[ip] = b

	if time.Since(rl.lastCleanup) > rl.cleanupEvery {
		rl.cleanup()
		rl.lastCleanup = time.Now()
	}

	return b
}

// cleanup removes buckets that have been idle for more than 10 minutes. Caller must hold rl.mu write lock.
func (rl *RateLimiter) cleanup() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, b := range rl.buckets {
		b.mu.Lock()
		idle := b.lastFill.Before(cutoff)
		b.mu.Unlock()
		if idle {
			delete(rl.buckets, ip)
		}
	}
}

// RateLimitMiddleware wraps next and returns 429 when the per-IP limit is exceeded.
func RateLimitMiddleware(next http.Handler, rl *RateLimiter, obs *observability.Obs) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.Allow(ip) {
			obs.Log.Warn(r.Context(), "rate limit exceeded",
				observability.F("ip", ip),
				observability.F("path", r.URL.Path),
			)
			w.Header().Set("Retry-After", "60")
			utils.WriteError(w, http.StatusTooManyRequests, "rate limit exceeded — try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP from common forwarding headers.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := len(xff); i > 0 {
			return xff[:i]
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr (strips port).
	addr := r.RemoteAddr
	if i := len(addr) - 1; i > 0 {
		for ; i >= 0 && addr[i] != ':'; i-- {
		}
		if i > 0 {
			return addr[:i]
		}
	}
	return addr
}

// HealthHandler responds to GET /health with a simple liveness check. The server is live if it can reach this handler.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"ts":     time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadyHandler responds to GET /ready. readyFn returns an error if the system is not ready to serve traffic (e.g. DB not connected, workers not started).
func ReadyHandler(readyFn func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := readyFn(); err != nil {
			utils.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not ready",
				"reason": err.Error(),
			})
			return
		}
		utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
