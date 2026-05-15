package providers

import (
	"errors"
)

var (
	ErrRateLimited      = errors.New("provider rate limited")
	ErrUnauthorized     = errors.New("provider unauthorized")
	ErrContextCancelled = errors.New("request cancelled")
	ErrTimeout          = errors.New("provider timeout")
)
