package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the current state of the circuit breaker
type State int

const (
	StateClosed   State = iota // Normal operation, requests flow through
	StateOpen                  // Circuit is open, requests are blocked
	StateHalfOpen              // Testing if the service has recovered
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker is the interface for circuit breaker operations
type CircuitBreaker interface {
	// Execute runs the given function if the circuit allows it
	Execute(fn func() (interface{}, error)) (interface{}, error)

	// State returns the current state of the circuit breaker
	State() State

	// Reset resets the circuit breaker to the initial closed state
	Reset()

	// Counts returns the current failure and success counts
	Counts() (failures int, successes int)
}

// Settings holds the configuration for the circuit breaker
type Settings struct {
	// FailureThreshold is the number of failures before the circuit opens
	FailureThreshold int

	// SuccessThreshold is the number of successes in half-open state before closing
	SuccessThreshold int

	// Timeout is the duration the circuit stays open before transitioning to half-open
	Timeout time.Duration

	// MaxRequests is the maximum number of requests allowed in half-open state
	MaxRequests int

	// OnStateChange is called when the circuit breaker state changes
	OnStateChange func(from, to State)
}

// DefaultSettings returns the default circuit breaker settings
func DefaultSettings() Settings {
	return Settings{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		MaxRequests:      1,
		OnStateChange:    nil,
	}
}

// circuitBreaker is the implementation of CircuitBreaker interface
type circuitBreaker struct {
	mu sync.RWMutex

	// Current state of the circuit breaker
	state State

	// Number of consecutive failures
	failureCount int

	// Number of consecutive successes (used in half-open state)
	successCount int

	// Number of requests in half-open state
	halfOpenRequests int

	// Time when the circuit was opened
	openedAt time.Time

	// Configuration settings
	settings Settings
}

// New creates a new circuit breaker with the given settings
func New(settings Settings) CircuitBreaker {
	return &circuitBreaker{
		state:    StateClosed,
		settings: settings,
	}
}

// NewWithDefaults creates a new circuit breaker with default settings
func NewWithDefaults() CircuitBreaker {
	return New(DefaultSettings())
}

// Execute runs the given function if the circuit allows it
func (cb *circuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	// Phase 1: Check if we can proceed
	cb.mu.Lock()

	if cb.state == StateOpen && time.Since(cb.openedAt) >= cb.settings.Timeout {
		cb.setState(StateHalfOpen)
		cb.halfOpenRequests = 0
		cb.successCount = 0
	}

	state := cb.state

	switch state {
	case StateOpen:
		cb.mu.Unlock()
		return nil, errors.New("circuit breaker is open")
	case StateHalfOpen:
		if cb.halfOpenRequests >= cb.settings.MaxRequests {
			cb.mu.Unlock()
			return nil, errors.New("circuit breaker is half-open")
		}
		cb.halfOpenRequests++
	}

	cb.mu.Unlock() // ✅ Unlock ก่อนเรียก fn()

	// Phase 2: Execute (without lock)
	result, err := fn()

	// Phase 3: Record result
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.successCount = 0
		// ถ้า fail ใน half-open หรือถึง threshold ให้กลับไป Open
		if state == StateHalfOpen || cb.failureCount >= cb.settings.FailureThreshold {
			cb.setState(StateOpen)
			cb.openedAt = time.Now()
		}
	} else {
		cb.successCount++
		cb.failureCount = 0
		if state == StateHalfOpen {
			cb.halfOpenRequests = 0 // Reset เพื่อให้ลองต่อได้
			if cb.successCount >= cb.settings.SuccessThreshold {
				cb.setState(StateClosed)
			}
		}
	}

	return result, err
}

// State returns the current state of the circuit breaker
func (cb *circuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to the initial closed state
func (cb *circuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenRequests = 0
	cb.openedAt = time.Time{}
}

// Counts returns the current failure and success counts
func (cb *circuitBreaker) Counts() (failures int, successes int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount, cb.successCount
}

func (cb *circuitBreaker) setState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	if cb.settings.OnStateChange != nil {
		cb.settings.OnStateChange(oldState, newState)
	}
}
