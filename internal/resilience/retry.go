package resilience

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxAttempts      int
	InitialDelay     time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       30 * time.Second,
		BackoffFactor:  2.0,
		RetryableErrors: func(err error) bool { return true },
	}
}

// Retry executes a function with retry logic
func Retry(fn func() error, config RetryConfig) error {
	return RetryWithContext(context.Background(), func(ctx context.Context) error {
		return fn()
	}, config)
}

// RetryWithContext executes a function with retry logic and context
func RetryWithContext(ctx context.Context, fn func(context.Context) error, config RetryConfig) error {
	var lastErr error
	
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		
		// Check if error is retryable
		if config.RetryableErrors != nil && !config.RetryableErrors(err) {
			return err
		}

		// Don't wait on the last attempt
		if attempt < config.MaxAttempts {
			delay := calculateDelay(attempt-1, config)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return fmt.Errorf("max retry attempts (%d) reached, last error: %w", config.MaxAttempts, lastErr)
}

// RetryWithResult executes a function with retry logic and returns result
func RetryWithResult[T any](fn func() (T, error), config RetryConfig) (T, error) {
	var zero T
	result, err := Retry(func() error {
		var err error
		result, err = fn()
		return err
	}, config)
	
	return result, err
}

// RetryWithContextAndResult executes a function with retry logic, context, and returns result
func RetryWithContextAndResult[T any](ctx context.Context, fn func(context.Context) (T, error), config RetryConfig) (T, error) {
	var zero T
	var result T
	
	err := RetryWithContext(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}, config)
	
	return result, err
}

// calculateDelay calculates the delay for a given attempt using exponential backoff
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt))
	
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	return time.Duration(delay)
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Common retryable errors
	errMsg := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"service unavailable",
		"rate limit exceeded",
		"circuit breaker open",
	}
	
	for _, retryableErr := range retryableErrors {
		if contains(errMsg, retryableErr) {
			return true
		}
	}
	
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     indexOf(s, substr) >= 0))
}

// indexOf finds the index of a substring in a string
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
