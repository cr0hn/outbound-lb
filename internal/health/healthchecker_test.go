package health

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockChecker is a mock implementation of Checker for testing.
type mockChecker struct {
	mu         sync.Mutex
	results    map[string]error // ip -> error (nil = success)
	checkCount atomic.Int64
}

func newMockChecker() *mockChecker {
	return &mockChecker{
		results: make(map[string]error),
	}
}

func (m *mockChecker) SetResult(ip string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results[ip] = err
}

func (m *mockChecker) Check(ctx context.Context, sourceIP string) error {
	m.checkCount.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.results[sourceIP]; ok {
		return err
	}
	return nil // default to success
}

func (m *mockChecker) GetCheckCount() int64 {
	return m.checkCount.Load()
}

func TestIPStatus_RecordSuccess(t *testing.T) {
	tests := []struct {
		name             string
		initialState     HealthState
		successThreshold int
		numSuccesses     int
		expectedState    HealthState
	}{
		{
			name:             "healthy stays healthy",
			initialState:     StateHealthy,
			successThreshold: 2,
			numSuccesses:     1,
			expectedState:    StateHealthy,
		},
		{
			name:             "unhealthy transitions to recovering on first success",
			initialState:     StateUnhealthy,
			successThreshold: 2,
			numSuccesses:     1,
			expectedState:    StateRecovering,
		},
		{
			name:             "recovering becomes healthy after threshold",
			initialState:     StateRecovering,
			successThreshold: 2,
			numSuccesses:     2,
			expectedState:    StateHealthy,
		},
		{
			name:             "recovering stays recovering before threshold",
			initialState:     StateRecovering,
			successThreshold: 3,
			numSuccesses:     1, // Only 1 success, need 3 to become healthy
			expectedState:    StateRecovering,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewIPStatus("192.168.1.1")
			status.State = tt.initialState
			// Don't pre-set consecutive successes - let RecordSuccess handle it

			for i := 0; i < tt.numSuccesses; i++ {
				status.RecordSuccess(tt.successThreshold)
			}

			if status.State != tt.expectedState {
				t.Errorf("expected state %v, got %v", tt.expectedState, status.State)
			}
		})
	}
}

func TestIPStatus_RecordFailure(t *testing.T) {
	tests := []struct {
		name             string
		initialState     HealthState
		failureThreshold int
		numFailures      int
		expectedState    HealthState
	}{
		{
			name:             "healthy becomes unhealthy after threshold",
			initialState:     StateHealthy,
			failureThreshold: 3,
			numFailures:      3,
			expectedState:    StateUnhealthy,
		},
		{
			name:             "healthy stays healthy before threshold",
			initialState:     StateHealthy,
			failureThreshold: 3,
			numFailures:      2,
			expectedState:    StateHealthy,
		},
		{
			name:             "recovering becomes unhealthy on any failure",
			initialState:     StateRecovering,
			failureThreshold: 3,
			numFailures:      1,
			expectedState:    StateUnhealthy,
		},
		{
			name:             "unhealthy stays unhealthy",
			initialState:     StateUnhealthy,
			failureThreshold: 3,
			numFailures:      1,
			expectedState:    StateUnhealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewIPStatus("192.168.1.1")
			status.State = tt.initialState

			testErr := errors.New("test error")
			for i := 0; i < tt.numFailures; i++ {
				status.RecordFailure(testErr, tt.failureThreshold)
			}

			if status.State != tt.expectedState {
				t.Errorf("expected state %v, got %v", tt.expectedState, status.State)
			}
		})
	}
}

func TestIPStatus_SuccessResetsFailures(t *testing.T) {
	status := NewIPStatus("192.168.1.1")

	// Record 2 failures (threshold is 3)
	testErr := errors.New("test error")
	status.RecordFailure(testErr, 3)
	status.RecordFailure(testErr, 3)

	if status.ConsecutiveFailures != 2 {
		t.Errorf("expected 2 consecutive failures, got %d", status.ConsecutiveFailures)
	}

	// Record a success - should reset failures
	status.RecordSuccess(2)

	if status.ConsecutiveFailures != 0 {
		t.Errorf("expected 0 consecutive failures after success, got %d", status.ConsecutiveFailures)
	}
}

func TestIPStatus_GetInfo(t *testing.T) {
	status := NewIPStatus("192.168.1.1")
	testErr := errors.New("connection refused")
	status.RecordFailure(testErr, 5)

	info := status.GetInfo()

	if info.IP != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", info.IP)
	}
	if info.State != "healthy" {
		t.Errorf("expected state healthy, got %s", info.State)
	}
	if info.ConsecutiveFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", info.ConsecutiveFailures)
	}
	if info.LastError != "connection refused" {
		t.Errorf("expected error 'connection refused', got %s", info.LastError)
	}
}

func TestHealthChecker_IsHealthy(t *testing.T) {
	checker := newMockChecker()
	hc := NewHealthChecker(HealthCheckerConfig{
		IPs:              []string{"192.168.1.1", "192.168.1.2"},
		Checker:          checker,
		Interval:         time.Hour, // Long interval so we control checks manually
		Timeout:          time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	})

	// All IPs should start healthy
	if !hc.IsHealthy("192.168.1.1") {
		t.Error("expected IP to be healthy initially")
	}
	if !hc.IsHealthy("192.168.1.2") {
		t.Error("expected IP to be healthy initially")
	}

	// Unknown IPs should be considered healthy
	if !hc.IsHealthy("10.0.0.1") {
		t.Error("expected unknown IP to be healthy")
	}
}

func TestHealthChecker_GetHealthyIPs(t *testing.T) {
	checker := newMockChecker()
	hc := NewHealthChecker(HealthCheckerConfig{
		IPs:              []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		Checker:          checker,
		Interval:         time.Hour,
		Timeout:          time.Second,
		FailureThreshold: 2,
		SuccessThreshold: 2,
	})

	// Mark one IP as unhealthy
	hc.mu.Lock()
	status := hc.statuses["192.168.1.2"]
	status.mu.Lock()
	status.State = StateUnhealthy
	status.mu.Unlock()
	hc.mu.Unlock()

	healthyIPs := hc.GetHealthyIPs([]string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})

	if len(healthyIPs) != 2 {
		t.Errorf("expected 2 healthy IPs, got %d", len(healthyIPs))
	}

	// Check that 192.168.1.2 is not in the result
	for _, ip := range healthyIPs {
		if ip == "192.168.1.2" {
			t.Error("unhealthy IP should not be in result")
		}
	}
}

func TestHealthChecker_CheckLoop(t *testing.T) {
	checker := newMockChecker()
	// Make 192.168.1.2 fail
	checker.SetResult("192.168.1.2", errors.New("connection refused"))

	hc := NewHealthChecker(HealthCheckerConfig{
		IPs:              []string{"192.168.1.1", "192.168.1.2"},
		Checker:          checker,
		Interval:         50 * time.Millisecond,
		Timeout:          time.Second,
		FailureThreshold: 2,
		SuccessThreshold: 2,
	})

	hc.Start()

	// Wait for enough checks to trigger unhealthy state
	time.Sleep(200 * time.Millisecond)

	hc.Stop()

	// 192.168.1.1 should be healthy
	if !hc.IsHealthy("192.168.1.1") {
		t.Error("expected 192.168.1.1 to be healthy")
	}

	// 192.168.1.2 should be unhealthy after 2 failures
	if hc.IsHealthy("192.168.1.2") {
		t.Error("expected 192.168.1.2 to be unhealthy")
	}
}

func TestHealthChecker_Recovery(t *testing.T) {
	checker := newMockChecker()
	// Start with 192.168.1.1 failing
	checker.SetResult("192.168.1.1", errors.New("connection refused"))

	hc := NewHealthChecker(HealthCheckerConfig{
		IPs:              []string{"192.168.1.1"},
		Checker:          checker,
		Interval:         30 * time.Millisecond,
		Timeout:          time.Second,
		FailureThreshold: 2,
		SuccessThreshold: 2,
	})

	hc.Start()

	// Wait for unhealthy
	time.Sleep(100 * time.Millisecond)

	if hc.IsHealthy("192.168.1.1") {
		t.Error("expected IP to be unhealthy")
	}

	// Fix the IP
	checker.SetResult("192.168.1.1", nil)

	// Wait for recovery (needs 2 successes)
	time.Sleep(150 * time.Millisecond)

	hc.Stop()

	// Should be healthy again
	if !hc.IsHealthy("192.168.1.1") {
		t.Error("expected IP to recover and be healthy")
	}
}

func TestHealthChecker_GetAllStatus(t *testing.T) {
	checker := newMockChecker()
	hc := NewHealthChecker(HealthCheckerConfig{
		IPs:              []string{"192.168.1.1", "192.168.1.2"},
		Checker:          checker,
		Interval:         time.Hour,
		Timeout:          time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	})

	statuses := hc.GetAllStatus()

	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}

	ipFound := make(map[string]bool)
	for _, s := range statuses {
		ipFound[s.IP] = true
		if s.State != "healthy" {
			t.Errorf("expected state healthy for %s, got %s", s.IP, s.State)
		}
	}

	if !ipFound["192.168.1.1"] || !ipFound["192.168.1.2"] {
		t.Error("expected both IPs in status")
	}
}

func TestHealthState_String(t *testing.T) {
	tests := []struct {
		state    HealthState
		expected string
	}{
		{StateHealthy, "healthy"},
		{StateUnhealthy, "unhealthy"},
		{StateRecovering, "recovering"},
		{HealthState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("HealthState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}
