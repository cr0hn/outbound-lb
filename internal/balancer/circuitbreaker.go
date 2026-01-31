// Package balancer provides IP load balancing algorithms.
package balancer

import (
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed means the circuit is closed and requests are allowed.
	StateClosed State = iota
	// StateOpen means the circuit is open and requests are rejected.
	StateOpen
	// StateHalfOpen means the circuit is testing if the IP is healthy again.
	StateHalfOpen
)

// String returns the string representation of the state.
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

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open state to close the circuit.
	SuccessThreshold int
	// Timeout is how long the circuit stays open before transitioning to half-open.
	Timeout time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}
}

// ipState holds the circuit breaker state for a single IP.
type ipState struct {
	failures    int
	successes   int
	state       State
	lastFailure time.Time
}

// CircuitBreaker manages circuit breaker state per IP.
type CircuitBreaker struct {
	mu     sync.RWMutex
	states map[string]*ipState
	config CircuitBreakerConfig
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		states: make(map[string]*ipState),
		config: config,
	}
}

// getOrCreateState returns the state for an IP, creating it if necessary.
func (cb *CircuitBreaker) getOrCreateState(ip string) *ipState {
	cb.mu.RLock()
	state, exists := cb.states[ip]
	cb.mu.RUnlock()

	if exists {
		return state
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Double-check after acquiring write lock
	if state, exists := cb.states[ip]; exists {
		return state
	}

	state = &ipState{state: StateClosed}
	cb.states[ip] = state
	return state
}

// IsHealthy checks if an IP is considered healthy (circuit not open).
// Returns true if requests should be allowed to this IP.
func (cb *CircuitBreaker) IsHealthy(ip string) bool {
	cb.mu.RLock()
	state, exists := cb.states[ip]
	cb.mu.RUnlock()

	if !exists {
		return true // No state means healthy
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch state.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(state.lastFailure) >= cb.config.Timeout {
			state.state = StateHalfOpen
			state.successes = 0
			return true // Allow one request to test
		}
		return false
	case StateHalfOpen:
		return true // Allow requests in half-open state
	default:
		return true
	}
}

// RecordSuccess records a successful request to an IP.
func (cb *CircuitBreaker) RecordSuccess(ip string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state, exists := cb.states[ip]
	if !exists {
		return // No state to update
	}

	switch state.state {
	case StateHalfOpen:
		state.successes++
		if state.successes >= cb.config.SuccessThreshold {
			// Close the circuit
			state.state = StateClosed
			state.failures = 0
			state.successes = 0
		}
	case StateClosed:
		// Reset failure count on success
		state.failures = 0
	}
}

// RecordFailure records a failed request to an IP.
func (cb *CircuitBreaker) RecordFailure(ip string) {
	state := cb.getOrCreateState(ip)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	state.lastFailure = time.Now()

	switch state.state {
	case StateClosed:
		state.failures++
		if state.failures >= cb.config.FailureThreshold {
			state.state = StateOpen
		}
	case StateHalfOpen:
		// Any failure in half-open opens the circuit again
		state.state = StateOpen
		state.successes = 0
	}
}

// GetState returns the current state for an IP.
func (cb *CircuitBreaker) GetState(ip string) State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, exists := cb.states[ip]
	if !exists {
		return StateClosed
	}
	return state.state
}

// GetStats returns statistics about all IPs.
func (cb *CircuitBreaker) GetStats() map[string]struct {
	State    State
	Failures int
} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := make(map[string]struct {
		State    State
		Failures int
	})

	for ip, state := range cb.states {
		stats[ip] = struct {
			State    State
			Failures int
		}{
			State:    state.state,
			Failures: state.failures,
		}
	}

	return stats
}

// Reset resets the circuit breaker state for an IP.
func (cb *CircuitBreaker) Reset(ip string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	delete(cb.states, ip)
}

// ResetAll resets all circuit breaker states.
func (cb *CircuitBreaker) ResetAll() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.states = make(map[string]*ipState)
}
