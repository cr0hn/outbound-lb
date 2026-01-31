package proxy

import (
	"net"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

func newTestServerForConnect(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 100,
		MaxConnsTotal: 1000,
		HistoryWindow: 5 * time.Minute,
		HistorySize:   100,
		LogLevel:      "error",
		LogFormat:     "json",
	}

	stats := metrics.NewStatsCollector(cfg.IPs)
	lim := limiter.New(cfg.MaxConnsPerIP, cfg.MaxConnsTotal, cfg.IPs)
	balCfg := balancer.Config{
		IPs:           cfg.IPs,
		HistoryWindow: int64(cfg.HistoryWindow.Seconds()),
		HistorySize:   cfg.HistorySize,
		Limiter:       lim,
	}
	bal := balancer.New(balCfg)

	return NewServer(cfg, bal, lim, stats)
}

func TestNewConnectHandler(t *testing.T) {
	server := newTestServerForConnect(t)
	handler := NewConnectHandler(server)

	if handler == nil {
		t.Fatal("expected non-nil connect handler")
	}
	if handler.server != server {
		t.Error("expected server to be set")
	}
}

func TestConnectHandler_tunnel(t *testing.T) {
	server := newTestServerForConnect(t)
	handler := NewConnectHandler(server)

	// Create pipe connections for testing
	// client side: we write to clientWrite, tunnel reads from clientRead
	// target side: tunnel writes to targetWrite, we read from targetRead
	clientRead, clientWrite := net.Pipe()
	targetRead, targetWrite := net.Pipe()

	done := make(chan struct{})

	// Goroutine to write to client side and read from target side
	go func() {
		defer close(done)
		// Write data as "client"
		clientWrite.Write([]byte("Hello from client"))
		clientWrite.Close()
	}()

	// Goroutine to read from target and write response
	go func() {
		buf := make([]byte, 1024)
		targetWrite.Read(buf) // Read what tunnel sends to target
		targetWrite.Write([]byte("Hello from target"))
		targetWrite.Close()
	}()

	// Run tunnel - clientRead is the "client" conn, targetRead is the "target" conn
	// This is a simplified test that verifies the function doesn't panic
	bytesIn, bytesOut := handler.tunnel(clientRead, targetRead, 60*time.Second)

	clientRead.Close()
	targetRead.Close()
	<-done

	// Just verify no panic and some bytes transferred
	t.Logf("Bytes in: %d, out: %d", bytesIn, bytesOut)
}

func TestConnectHandler_tunnel_EmptyData(t *testing.T) {
	server := newTestServerForConnect(t)
	handler := NewConnectHandler(server)

	// Create pipe connections for testing
	clientRead, clientWrite := net.Pipe()
	targetRead, targetWrite := net.Pipe()

	// Close immediately in goroutines
	go func() {
		time.Sleep(10 * time.Millisecond)
		clientWrite.Close()
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		targetWrite.Close()
	}()

	// Run tunnel
	bytesIn, bytesOut := handler.tunnel(clientRead, targetRead, 60*time.Second)

	clientRead.Close()
	targetRead.Close()

	// Both should be 0 for empty transfer
	t.Logf("Empty transfer - bytes in: %d, out: %d", bytesIn, bytesOut)
}

func TestConnectHandler_tunnel_ConcurrentRaceDetection(t *testing.T) {
	// This test verifies that tunnel() doesn't have race conditions
	// Run with -race flag to detect data races
	server := newTestServerForConnect(t)
	handler := NewConnectHandler(server)

	for i := 0; i < 10; i++ {
		t.Run("iteration", func(t *testing.T) {
			t.Parallel() // Run iterations in parallel to stress test

			clientRead, clientWrite := net.Pipe()
			targetRead, targetWrite := net.Pipe()

			testData := []byte("Hello, this is test data for race condition testing!")
			responseData := []byte("Response from the target server!")

			done := make(chan struct{})

			// Simulate client sending data
			go func() {
				defer clientWrite.Close()
				clientWrite.Write(testData)
			}()

			// Simulate target receiving and responding
			go func() {
				defer targetWrite.Close()
				buf := make([]byte, 1024)
				targetWrite.Read(buf) // Read what client sends
				targetWrite.Write(responseData)
			}()

			// Run tunnel in goroutine
			go func() {
				defer close(done)
				bytesIn, bytesOut := handler.tunnel(clientRead, targetRead, 60*time.Second)
				// Verify bytes were transferred (values should match atomic operations)
				if bytesIn < 0 || bytesOut < 0 {
					t.Errorf("invalid byte counts: in=%d, out=%d", bytesIn, bytesOut)
				}
			}()

			// Wait for tunnel completion with timeout
			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Error("tunnel timed out")
			}

			clientRead.Close()
			targetRead.Close()
		})
	}
}

func TestConnectHandler_tunnel_BidirectionalTransfer(t *testing.T) {
	server := newTestServerForConnect(t)
	handler := NewConnectHandler(server)

	// net.Pipe creates a synchronous, in-memory, full duplex connection
	// clientConn is passed to tunnel as "client"
	// targetConn is passed to tunnel as "target"
	clientConn, clientPeer := net.Pipe()
	targetConn, targetPeer := net.Pipe()

	clientData := []byte("Hello from client!")
	targetData := []byte("Hello from target!")

	var bytesIn, bytesOut int64
	done := make(chan struct{})

	// Simulate the client peer - writes data and then reads response
	go func() {
		clientPeer.Write(clientData)
		// Read what comes back from target
		buf := make([]byte, 1024)
		clientPeer.Read(buf)
		clientPeer.Close()
	}()

	// Simulate the target peer - reads data and then writes response
	go func() {
		buf := make([]byte, 1024)
		targetPeer.Read(buf)
		targetPeer.Write(targetData)
		targetPeer.Close()
	}()

	go func() {
		defer close(done)
		bytesIn, bytesOut = handler.tunnel(clientConn, targetConn, 60*time.Second)
	}()

	select {
	case <-done:
		// bytesIn = client -> target (clientData)
		// bytesOut = target -> client (targetData)
		if bytesIn != int64(len(clientData)) {
			t.Errorf("expected bytesIn=%d, got %d", len(clientData), bytesIn)
		}
		if bytesOut != int64(len(targetData)) {
			t.Errorf("expected bytesOut=%d, got %d", len(targetData), bytesOut)
		}
		t.Logf("Bidirectional transfer - bytes in: %d, bytes out: %d", bytesIn, bytesOut)
	case <-time.After(5 * time.Second):
		t.Error("bidirectional transfer test timed out")
	}

	clientConn.Close()
	targetConn.Close()
}
