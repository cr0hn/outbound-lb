package health

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPChecker_Check_Success(t *testing.T) {
	// Start a local TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Accept connections in goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	checker := NewTCPChecker(listener.Addr().String(), 5*time.Second)

	ctx := context.Background()
	err = checker.Check(ctx, "127.0.0.1")

	if err != nil {
		t.Errorf("expected check to succeed, got error: %v", err)
	}
}

func TestTCPChecker_Check_Failure(t *testing.T) {
	// Use a port that should not have anything listening
	checker := NewTCPChecker("127.0.0.1:59999", 1*time.Second)

	ctx := context.Background()
	err := checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected check to fail, but it succeeded")
	}
}

func TestTCPChecker_Check_Timeout(t *testing.T) {
	// Use a non-routable IP to cause timeout
	checker := NewTCPChecker("10.255.255.1:80", 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := checker.Check(ctx, "127.0.0.1")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected check to fail due to timeout")
	}

	// Should timeout relatively quickly
	if elapsed > 500*time.Millisecond {
		t.Errorf("check took too long: %v", elapsed)
	}
}

func TestTCPChecker_Check_ContextCancellation(t *testing.T) {
	// Start a server that delays accepting
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	checker := NewTCPChecker(listener.Addr().String(), 5*time.Second)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = checker.Check(ctx, "127.0.0.1")

	if err == nil {
		t.Error("expected check to fail due to context cancellation")
	}
}
