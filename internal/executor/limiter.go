package executor

import (
	"context"
	"sync"
	"time"
)

// Limiter defines the interface for rate limiting.
type Limiter interface {
	Wait(ctx context.Context) error
	Allow() bool
}

// TokenBucketLimiter implements the token bucket algorithm.
type TokenBucketLimiter struct {
	rate       float64 // tokens per second
	capacity   float64
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucketLimiter(rate, capacity float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
	}
}

func (l *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.tokens += elapsed * l.rate
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	l.lastRefill = now
}

func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		l.refill()
		if l.tokens >= 1 {
			l.tokens -= 1
			l.mu.Unlock()
			return nil
		}

		// Calculate sleep time
		needed := 1 - l.tokens
		sleepTime := time.Duration(needed / l.rate * float64(time.Second))
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepTime):
			// Try again
		}
	}
}

func (l *TokenBucketLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	if l.tokens >= 1 {
		l.tokens -= 1
		return true
	}
	return false
}

// ConcurrencyLimiter limits the number of concurrent executions using a semaphore.
type ConcurrencyLimiter struct {
	sem chan struct{}
}

func NewConcurrencyLimiter(limit int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		sem: make(chan struct{}, limit),
	}
}

func (l *ConcurrencyLimiter) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.sem <- struct{}{}:
		return nil
	}
}

func (l *ConcurrencyLimiter) Release() {
	<-l.sem
}
