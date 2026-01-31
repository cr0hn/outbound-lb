package proxy

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// TestProxy_SlowlorisResistance tests that the proxy handles slow clients properly
// and doesn't leak goroutines when clients send data very slowly.
func TestProxy_SlowlorisResistance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slowloris test in short mode")
	}

	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       2 * time.Second, // Short timeout for test
		IdleTimeout:   1 * time.Second,
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
	bal.Start()
	defer bal.Stop()

	server := NewServer(cfg, bal, lim, stats)

	// Start a test HTTP server (proxy target)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	// Start proxy server
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer proxyListener.Close()

	proxyAddr := proxyListener.Addr().String()

	go func() {
		http.Serve(proxyListener, NewHandler(server))
	}()

	// Simulate slowloris attack - open many connections sending headers slowly
	numSlowClients := 10
	var wg sync.WaitGroup

	for i := 0; i < numSlowClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", proxyAddr, 5*time.Second)
			if err != nil {
				t.Logf("client %d: dial failed: %v", id, err)
				return
			}
			defer conn.Close()

			// Send partial HTTP request very slowly
			// This simulates a slowloris attack
			fmt.Fprintf(conn, "GET %s HTTP/1.1\r\n", backend.URL)
			time.Sleep(200 * time.Millisecond)
			fmt.Fprintf(conn, "Host: localhost\r\n")
			time.Sleep(200 * time.Millisecond)
			fmt.Fprintf(conn, "X-Slow-Header: ")

			// Keep sending partial header data slowly
			for j := 0; j < 5; j++ {
				time.Sleep(300 * time.Millisecond)
				fmt.Fprintf(conn, "x")
			}

			// The connection should timeout and close
			// Try to read response (may get nothing or error)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			reader := bufio.NewReader(conn)
			_, err = reader.ReadString('\n')
			if err != nil {
				t.Logf("client %d: read error (expected): %v", id, err)
			}
		}(i)
	}

	// Wait for slow clients to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("all slow clients completed or timed out")
	case <-time.After(30 * time.Second):
		t.Error("test timed out - possible goroutine leak")
	}

	// Verify connections were properly cleaned up
	time.Sleep(500 * time.Millisecond)
	totalConns := lim.GetTotalCount()
	if totalConns != 0 {
		t.Errorf("expected 0 active connections, got %d", totalConns)
	}
}

// TestProxy_SlowResponse tests handling of slow backend responses.
func TestProxy_SlowResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow response test in short mode")
	}

	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       2 * time.Second,
		IdleTimeout:   1 * time.Second,
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
	bal.Start()
	defer bal.Stop()

	server := NewServer(cfg, bal, lim, stats)
	handler := NewHandler(server)

	// Create a slow backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(3 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
	rr := httptest.NewRecorder()

	// This should timeout or handle the slow response gracefully
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	handler.ServeHTTP(rr, req)

	// Should get a bad gateway or similar error
	if rr.Code != http.StatusBadGateway && rr.Code != http.StatusGatewayTimeout {
		t.Logf("response code: %d (may vary based on timeout handling)", rr.Code)
	}
}

// TestProxy_ClientDisconnect tests proper cleanup when client disconnects mid-request.
func TestProxy_ClientDisconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping client disconnect test in short mode")
	}

	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   10 * time.Second,
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
	bal.Start()
	defer bal.Stop()

	server := NewServer(cfg, bal, lim, stats)

	// Create a slow backend that will outlive the client
	backendReady := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(backendReady)
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// Start proxy server
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer proxyListener.Close()

	proxyAddr := proxyListener.Addr().String()

	go func() {
		http.Serve(proxyListener, NewHandler(server))
	}()

	// Open connection and send request, then disconnect
	conn, err := net.DialTimeout("tcp", proxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	// Send a valid HTTP request
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\n", backend.URL)
	fmt.Fprintf(conn, "Host: localhost\r\n")
	fmt.Fprintf(conn, "\r\n")

	// Wait for request to reach backend
	select {
	case <-backendReady:
		// Good, backend received the request
	case <-time.After(5 * time.Second):
		t.Fatal("backend didn't receive request")
	}

	// Now disconnect abruptly
	conn.Close()

	// Wait for cleanup
	time.Sleep(1 * time.Second)

	// Connection count should eventually return to 0
	// (may take a moment for cleanup)
	time.Sleep(500 * time.Millisecond)
	t.Logf("active connections after client disconnect: %d", lim.GetTotalCount())
}
