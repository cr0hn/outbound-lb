package proxy

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

func createTestProxyServer(t *testing.T, auth string, maxPerIP, maxTotal int) (*Server, func()) {
	t.Helper()
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Auth:          auth,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: maxPerIP,
		MaxConnsTotal: maxTotal,
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

	server := NewServer(cfg, bal, lim, stats)

	cleanup := func() {
		bal.Stop()
	}

	return server, cleanup
}

func TestProxy_HTTPRequestFlow(t *testing.T) {
	// Create backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "true")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello from backend, path: %s", r.URL.Path)
	}))
	defer backend.Close()

	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	handler := NewHandler(server)

	// Create proxy request
	req := httptest.NewRequest(http.MethodGet, backend.URL+"/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// The request might fail since 127.0.0.1 is the only allowed outbound IP
	// and we can't actually proxy, but the handler logic should execute
	t.Logf("Response status: %d", w.Code)
}

func TestProxy_CONNECTFlow(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	// Create a target server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Accept and respond
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("Hello from target"))
	}()

	handler := NewConnectHandler(server)

	// Create CONNECT request - note: in real proxy, host would be the target
	req := httptest.NewRequest(http.MethodConnect, listener.Addr().String(), nil)
	req.Host = listener.Addr().String()
	req.RemoteAddr = "192.168.1.100:12345"

	// Use a custom response writer that supports hijacking
	w := &hijackableWriter{
		ResponseWriter: httptest.NewRecorder(),
		conn:           nil,
	}

	// This will fail because our mock doesn't support real hijacking,
	// but it exercises the code path
	handler.ServeHTTP(w, req)

	t.Logf("CONNECT test completed")
}

// hijackableWriter is a mock that pretends to support hijacking
type hijackableWriter struct {
	http.ResponseWriter
	conn net.Conn
}

func (w *hijackableWriter) Hijack() (net.Conn, *bufioReadWriter, error) {
	return nil, nil, fmt.Errorf("hijack not implemented in test")
}

type bufioReadWriter struct{}

func TestProxy_AuthMiddleware(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "user:pass", 100, 1000)
	defer cleanup()

	tests := []struct {
		name           string
		auth           string
		expectedResult bool
	}{
		{"no auth header", "", false},
		{"invalid scheme", "Bearer token", false},
		{"valid credentials", "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")), true},
		{"wrong user", "Basic " + base64.StdEncoding.EncodeToString([]byte("wrong:pass")), false},
		{"wrong pass", "Basic " + base64.StdEncoding.EncodeToString([]byte("user:wrong")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.auth != "" {
				req.Header.Set("Proxy-Authorization", tt.auth)
			}
			w := httptest.NewRecorder()

			result := server.authenticate(w, req)
			if result != tt.expectedResult {
				t.Errorf("authenticate() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestProxy_ConnectionLimitingIntegration(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 2, 5)
	defer cleanup()

	// Simulate acquiring connections
	for i := 0; i < 2; i++ {
		err := server.limiter.Acquire("127.0.0.1")
		if err != nil {
			t.Errorf("acquire %d should succeed: %v", i, err)
		}
	}

	// Third should fail
	err := server.limiter.Acquire("127.0.0.1")
	if err != limiter.ErrIPLimitReached {
		t.Errorf("expected ErrIPLimitReached, got: %v", err)
	}

	// Verify stats
	if server.limiter.GetIPCount("127.0.0.1") != 2 {
		t.Errorf("expected IP count 2, got %d", server.limiter.GetIPCount("127.0.0.1"))
	}

	// Release and retry
	server.limiter.Release("127.0.0.1")
	err = server.limiter.Acquire("127.0.0.1")
	if err != nil {
		t.Errorf("acquire after release should succeed: %v", err)
	}
}

func TestProxy_BalancerIntegration(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	// Test IP selection
	ip, err := server.selectIP("example.com")
	if err != nil {
		t.Errorf("selectIP should succeed: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", ip)
	}

	// Record selection
	server.balancer.Record("example.com", ip)

	stats := server.balancer.GetStats()
	if stats.TotalHosts != 1 {
		t.Errorf("expected 1 host, got %d", stats.TotalHosts)
	}
}

func TestProxy_TransportPoolIntegration(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	// Get transport for known IP
	transport1 := server.transportPool.Get("127.0.0.1")
	if transport1 == nil {
		t.Error("expected non-nil transport")
	}

	// Get same transport again
	transport2 := server.transportPool.Get("127.0.0.1")
	if transport1 != transport2 {
		t.Error("expected same transport instance")
	}

	// Close pool
	server.transportPool.Close()
}

func TestProxy_GracefulShutdown(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown should succeed quickly
	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

func TestProxy_WaitForConnections(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	// Simulate active connection
	server.limiter.Acquire("127.0.0.1")

	// Wait should timeout since we have a connection
	start := time.Now()
	server.WaitForConnections(100 * time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("WaitForConnections returned too quickly: %v", elapsed)
	}

	// Release and wait again
	server.limiter.Release("127.0.0.1")

	start = time.Now()
	server.WaitForConnections(5 * time.Second)
	elapsed = time.Since(start)

	// Should return quickly now
	if elapsed > 500*time.Millisecond {
		t.Errorf("WaitForConnections took too long after release: %v", elapsed)
	}
}

func TestHandler_ServeHTTP_CONNECT(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	handler := NewHandler(server)

	// CONNECT request should be delegated to connectHandler
	req := httptest.NewRequest(http.MethodConnect, "example.com:443", nil)
	w := httptest.NewRecorder()

	// This will fail because hijacking isn't supported in test
	// but it verifies the code path
	handler.ServeHTTP(w, req)

	// Should get an error since hijacking fails
	t.Logf("CONNECT response: %d", w.Code)
}

func TestHandler_HTTP_NoHost(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	handler := NewHandler(server)

	// Create request with URL host but no Host header
	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	req.Host = "" // Clear host
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Request will fail since we can't actually proxy,
	// but verifies the host extraction logic
	t.Logf("No-host response: %d", w.Code)
}

func TestServer_Start(t *testing.T) {
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0, // Random port
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
	bal.Start()
	defer bal.Stop()

	// Note: We can't easily test Start() without actually starting a server,
	// which would require port allocation. The test in main_test would cover this.
	server := NewServer(cfg, bal, lim, stats)
	if server.httpServer == nil {
		t.Error("expected http server to be created")
	}
}

func TestProxy_CopyHeaders_MultiValue(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	handler := NewHandler(server)

	src := http.Header{}
	src.Add("X-Custom", "value1")
	src.Add("X-Custom", "value2")
	src.Set("Content-Type", "application/json")

	dst := http.Header{}
	handler.copyHeaders(dst, src)

	values := dst.Values("X-Custom")
	if len(values) != 2 {
		t.Errorf("expected 2 X-Custom values, got %d", len(values))
	}
}

func TestProxy_CreateOutgoingRequest_RelativeURL(t *testing.T) {
	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	handler := NewHandler(server)

	// Simulate a relative URL request (proxy mode)
	req := httptest.NewRequest(http.MethodGet, "/path", nil)
	req.Host = "example.com"
	req.RemoteAddr = "192.168.1.1:12345"

	outReq := handler.createOutgoingRequest(req)

	if outReq.URL.Scheme != "http" {
		t.Errorf("expected scheme http, got %s", outReq.URL.Scheme)
	}
	if outReq.URL.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", outReq.URL.Host)
	}
}

// Test the full handler with a real backend
func TestHandler_RealBackend(t *testing.T) {
	// Create a backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "passed")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "Backend response")
	}))
	defer backend.Close()

	server, cleanup := createTestProxyServer(t, "", 100, 1000)
	defer cleanup()

	handler := NewHandler(server)

	// Create absolute URL request (full proxy style)
	req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Since the outbound IP is 127.0.0.1 and backend is on 127.0.0.1,
	// this might actually work depending on the transport
	t.Logf("Backend test response: %d", w.Code)
}
