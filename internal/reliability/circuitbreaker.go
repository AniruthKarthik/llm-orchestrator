package reliability

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

type Config struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	OnStateChange    func(name string, from, to State)
}

type CircuitBreaker struct {
	cfg          Config
	mu           sync.RWMutex
	state        State
	failures     int
	successes    int
	lastFailure  time.Time
}

func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		cfg:   cfg,
		state: StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !cb.allow() {
		return fmt.Errorf("circuit breaker %q is open", cb.cfg.Name)
	}

	err := fn(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateClosed {
		return true
	}

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.cfg.Timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.setState(StateHalfOpen)
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	}

	return true // Half-open allows some requests
}

func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		cb.setState(StateOpen)
	} else if cb.failures >= cb.cfg.FailureThreshold {
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) onSuccess() {
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.cfg.SuccessThreshold {
			cb.setState(StateClosed)
		}
	} else if cb.state == StateClosed {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) setState(s State) {
	if cb.state == s {
		return
	}

	old := cb.state
	cb.state = s
	cb.failures = 0
	cb.successes = 0

	if cb.cfg.OnStateChange != nil {
		cb.cfg.OnStateChange(cb.cfg.Name, old, s)
	}
}
