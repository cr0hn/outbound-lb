package balancer

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	// New IP should be healthy
	if !cb.IsHealthy("192.168.1.1") {
		t.Error("new IP should be healthy")
	}

	// State should be closed
	if state := cb.GetState("192.168.1.1"); state != StateClosed {
		t.Errorf("expected StateClosed, got %s", state)
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)
	ip := "192.168.1.1"

	// Record failures up to threshold
	for i := 0; i < 3; i++ {
		cb.RecordFailure(ip)
	}

	// Circuit should now be open
	if cb.IsHealthy(ip) {
		t.Error("circuit should be open after threshold failures")
	}
	if state := cb.GetState(ip); state != StateOpen {
		t.Errorf("expected StateOpen, got %s", state)
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)
	ip := "192.168.1.1"

	// Record some failures
	cb.RecordFailure(ip)
	cb.RecordFailure(ip)

	// Record success
	cb.RecordSuccess(ip)

	// Should still be healthy
	if !cb.IsHealthy(ip) {
		t.Error("circuit should be healthy after success")
	}

	// Record more failures - shouldn't open because failures were reset
	cb.RecordFailure(ip)
	cb.RecordFailure(ip)

	if !cb.IsHealthy(ip) {
		t.Error("circuit should still be healthy - failures were reset")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)
	ip := "192.168.1.1"

	// Open the circuit
	cb.RecordFailure(ip)
	cb.RecordFailure(ip)

	if cb.IsHealthy(ip) {
		t.Error("circuit should be open")
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open and allow request
	if !cb.IsHealthy(ip) {
		t.Error("circuit should be half-open and allow request")
	}
	if state := cb.GetState(ip); state != StateHalfOpen {
		t.Errorf("expected StateHalfOpen, got %s", state)
	}
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)
	ip := "192.168.1.1"

	// Open the circuit
	cb.RecordFailure(ip)
	cb.RecordFailure(ip)

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.IsHealthy(ip) // This triggers the transition

	// Record successes in half-open
	cb.RecordSuccess(ip)
	cb.RecordSuccess(ip)

	// Circuit should be closed now
	if state := cb.GetState(ip); state != StateClosed {
		t.Errorf("expected StateClosed after successes, got %s", state)
	}
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)
	ip := "192.168.1.1"

	// Open the circuit
	cb.RecordFailure(ip)
	cb.RecordFailure(ip)

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.IsHealthy(ip) // This triggers the transition

	// Fail in half-open state
	cb.RecordFailure(ip)

	// Circuit should be open again
	if state := cb.GetState(ip); state != StateOpen {
		t.Errorf("expected StateOpen after failure in half-open, got %s", state)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	ip := "192.168.1.1"

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure(ip)
	}

	if cb.IsHealthy(ip) {
		t.Error("circuit should be open")
	}

	// Reset
	cb.Reset(ip)

	// Should be healthy again
	if !cb.IsHealthy(ip) {
		t.Error("circuit should be healthy after reset")
	}
}

func TestCircuitBreaker_ResetAll(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	// Open circuits for multiple IPs
	for i := 0; i < 3; i++ {
		ip := "192.168.1." + string(rune('1'+i))
		for j := 0; j < 5; j++ {
			cb.RecordFailure(ip)
		}
	}

	cb.ResetAll()

	// All should be healthy
	for i := 0; i < 3; i++ {
		ip := "192.168.1." + string(rune('1'+i))
		if !cb.IsHealthy(ip) {
			t.Errorf("IP %s should be healthy after reset all", ip)
		}
	}
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 10,
		SuccessThreshold: 5,
		Timeout:          100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)
	ip := "192.168.1.1"

	var wg sync.WaitGroup

	// Multiple goroutines recording failures and successes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cb.IsHealthy(ip)
				if j%2 == 0 {
					cb.RecordFailure(ip)
				} else {
					cb.RecordSuccess(ip)
				}
			}
		}(i)
	}

	wg.Wait()

	// Just verify no panic or deadlock
	_ = cb.GetStats()
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())

	cb.RecordFailure("192.168.1.1")
	cb.RecordFailure("192.168.1.1")
	cb.RecordFailure("192.168.1.2")

	stats := cb.GetStats()

	if len(stats) != 2 {
		t.Errorf("expected 2 IPs in stats, got %d", len(stats))
	}

	if stats["192.168.1.1"].Failures != 2 {
		t.Errorf("expected 2 failures for 192.168.1.1, got %d", stats["192.168.1.1"].Failures)
	}
	if stats["192.168.1.2"].Failures != 1 {
		t.Errorf("expected 1 failure for 192.168.1.2, got %d", stats["192.168.1.2"].Failures)
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}
