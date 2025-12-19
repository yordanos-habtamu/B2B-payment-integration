package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrServiceUnavailable   = errors.New("service unavailable")
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name           string
	maxFailures    int
	resetTimeout   time.Duration
	requestTimeout time.Duration

	state        CircuitState
	failures     int
	lastFailTime  time.Time
	mutex        sync.RWMutex
	onStateChange func(name string, from, to CircuitState)
}

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name           string
	MaxFailures    int
	ResetTimeout   time.Duration
	RequestTimeout time.Duration
	OnStateChange  func(name string, from, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 30 * time.Second
	}

	return &CircuitBreaker{
		name:           config.Name,
		maxFailures:    config.MaxFailures,
		resetTimeout:   config.ResetTimeout,
		requestTimeout: config.RequestTimeout,
		state:          StateClosed,
		onStateChange:  config.OnStateChange,
	}
}

// Execute runs the given function through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.ExecuteWithContext(context.Background(), fn)
}

// ExecuteWithContext runs the given function through the circuit breaker with context
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitBreakerOpen
	}

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, cb.requestTimeout)
	defer cancel()

	// Execute the function
	err := fn()
	cb.recordResult(err == nil)

	if err != nil {
		return err
	}

	return nil
}

// ExecuteWithResult runs the given function and returns both result and error
func (cb *CircuitBreaker) ExecuteWithResult[T any](fn func() (T, error)) (T, error) {
	var zero T
	err := cb.Execute(func() error {
		_, err := fn()
		return err
	})
	if err != nil {
		return zero, err
	}
	
	result, err := fn()
	return result, err
}

// ExecuteWithContextAndResult runs the given function with context and returns both result and error
func (cb *CircuitBreaker) ExecuteWithContextAndResult[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	
	if !cb.allowRequest() {
		return zero, ErrCircuitBreakerOpen
	}

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, cb.requestTimeout)
	defer cancel()

	// Execute the function
	result, err := fn(timeoutCtx)
	cb.recordResult(err == nil)

	return result, err
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		return time.Since(cb.lastFailTime) >= cb.resetTimeout
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.failures = 0
	
	if cb.state == StateHalfOpen {
		cb.setState(StateClosed)
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.setState(StateOpen)
	}
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state != newState {
		oldState := cb.state
		cb.state = newState
		
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, newState)
		}
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Failures returns the current number of failures
func (cb *CircuitBreaker) Failures() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failures
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.setState(StateClosed)
	cb.failures = 0
	cb.lastFailTime = time.Time{}
}

// CircuitBreakerRegistry manages multiple circuit breakers
type CircuitBreakerRegistry struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Register registers a new circuit breaker
func (r *CircuitBreakerRegistry) Register(config CircuitBreakerConfig) *CircuitBreaker {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	cb := NewCircuitBreaker(config)
	r.breakers[config.Name] = cb
	return cb
}

// Get returns a circuit breaker by name
func (r *CircuitBreakerRegistry) Get(name string) (*CircuitBreaker, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	cb, exists := r.breakers[name]
	return cb, exists
}

// GetAll returns all circuit breakers
func (r *CircuitBreakerRegistry) GetAll() map[string]*CircuitBreaker {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	result := make(map[string]*CircuitBreaker)
	for name, cb := range r.breakers {
		result[name] = cb
	}
	return result
}

// ResetAll resets all circuit breakers
func (r *CircuitBreakerRegistry) ResetAll() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, cb := range r.breakers {
		cb.Reset()
	}
}

// GetStats returns statistics for all circuit breakers
func (r *CircuitBreakerRegistry) GetStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	for name, cb := range r.breakers {
		stats[name] = map[string]interface{}{
			"state":    cb.State(),
			"failures":  cb.Failures(),
		}
	}
	return stats
}
