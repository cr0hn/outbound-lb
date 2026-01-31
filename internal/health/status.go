// Package health provides IP health checking functionality.
package health

import (
	"sync"
	"time"
)

// HealthState represents the health state of an IP.
type HealthState int

const (
	// StateHealthy means the IP is healthy and can be used.
	StateHealthy HealthState = iota
	// StateUnhealthy means the IP has failed health checks and should not be used.
	StateUnhealthy
	// StateRecovering means the IP is being tested after being unhealthy.
	StateRecovering
)

// String returns a human-readable representation of the health state.
func (s HealthState) String() string {
	switch s {
	case StateHealthy:
		return "healthy"
	case StateUnhealthy:
		return "unhealthy"
	case StateRecovering:
		return "recovering"
	default:
		return "unknown"
	}
}

// IPStatus holds the current health status of a single IP.
type IPStatus struct {
	IP                   string
	State                HealthState
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	LastCheck            time.Time
	LastError            error
	mu                   sync.RWMutex
}

// NewIPStatus creates a new IPStatus for the given IP.
func NewIPStatus(ip string) *IPStatus {
	return &IPStatus{
		IP:    ip,
		State: StateHealthy,
	}
}

// GetState returns the current health state.
func (s *IPStatus) GetState() HealthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// IsHealthy returns true if the IP is in a healthy state.
func (s *IPStatus) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == StateHealthy
}

// RecordSuccess records a successful health check.
// Returns true if state changed.
func (s *IPStatus) RecordSuccess(successThreshold int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastCheck = time.Now()
	s.LastError = nil
	s.ConsecutiveFailures = 0
	s.ConsecutiveSuccesses++

	oldState := s.State

	switch s.State {
	case StateUnhealthy:
		// First success after being unhealthy -> recovering
		s.State = StateRecovering
		s.ConsecutiveSuccesses = 1
	case StateRecovering:
		// Need successThreshold consecutive successes to become healthy
		if s.ConsecutiveSuccesses >= successThreshold {
			s.State = StateHealthy
		}
	case StateHealthy:
		// Already healthy, nothing to do
	}

	return oldState != s.State
}

// RecordFailure records a failed health check.
// Returns true if state changed.
func (s *IPStatus) RecordFailure(err error, failureThreshold int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastCheck = time.Now()
	s.LastError = err
	s.ConsecutiveSuccesses = 0
	s.ConsecutiveFailures++

	oldState := s.State

	switch s.State {
	case StateHealthy:
		// Need failureThreshold consecutive failures to become unhealthy
		if s.ConsecutiveFailures >= failureThreshold {
			s.State = StateUnhealthy
		}
	case StateRecovering:
		// Any failure while recovering goes back to unhealthy
		s.State = StateUnhealthy
	case StateUnhealthy:
		// Already unhealthy, nothing to do
	}

	return oldState != s.State
}

// GetInfo returns a copy of the status info for external use.
func (s *IPStatus) GetInfo() StatusInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lastErr string
	if s.LastError != nil {
		lastErr = s.LastError.Error()
	}

	return StatusInfo{
		IP:                   s.IP,
		State:                s.State.String(),
		ConsecutiveFailures:  s.ConsecutiveFailures,
		ConsecutiveSuccesses: s.ConsecutiveSuccesses,
		LastCheck:            s.LastCheck,
		LastError:            lastErr,
	}
}

// StatusInfo is a serializable representation of IPStatus.
type StatusInfo struct {
	IP                   string    `json:"ip"`
	State                string    `json:"state"`
	ConsecutiveFailures  int       `json:"consecutive_failures"`
	ConsecutiveSuccesses int       `json:"consecutive_successes"`
	LastCheck            time.Time `json:"last_check"`
	LastError            string    `json:"last_error,omitempty"`
}
