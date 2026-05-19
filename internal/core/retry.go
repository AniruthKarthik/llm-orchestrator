package core

import "time"

// FailurePolicy defines how to handle failures.
type FailurePolicy string

const (
	// FailurePolicyFailFast immediately fails the workflow or stage.
	FailurePolicyFailFast FailurePolicy = "FAIL_FAST"
	// FailurePolicyContinueOnFailure continues execution despite the failure.
	FailurePolicyContinueOnFailure FailurePolicy = "CONTINUE_ON_FAILURE"
)

// RetryStrategy defines the strategy for calculating retry delays.
type RetryStrategy string

const (
	// RetryStrategyImmediate retries immediately without delay.
	RetryStrategyImmediate RetryStrategy = "IMMEDIATE"
	// RetryStrategyFixed applies a fixed delay between retries.
	RetryStrategyFixed RetryStrategy = "FIXED"
	// RetryStrategyExponential applies an exponential backoff between retries.
	RetryStrategyExponential RetryStrategy = "EXPONENTIAL"
)

// RetryPolicy defines the rules for retrying a failed task or workflow.
type RetryPolicy struct {
	MaxRetries    int           `json:"maxRetries"`
	Strategy      RetryStrategy `json:"strategy"`
	InitialDelay  time.Duration `json:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
}

// CalculateDelay computes the delay before the next retry based on the attempt number.
func (r RetryPolicy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 || r.MaxRetries == 0 || r.Strategy == RetryStrategyImmediate {
		return 0
	}

	delay := r.InitialDelay

	if r.Strategy == RetryStrategyExponential {
		for i := 1; i < attempt; i++ {
			delay = time.Duration(float64(delay) * r.BackoffFactor)
		}
	}

	if r.MaxDelay > 0 && delay > r.MaxDelay {
		return r.MaxDelay
	}

	return delay
}
