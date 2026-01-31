package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// TestServerOptions holds options for creating test servers.
type TestServerOptions struct {
	IPs           []string
	Timeout       time.Duration
	IdleTimeout   time.Duration
	MaxConnsPerIP int
	MaxConnsTotal int
	HistoryWindow time.Duration
	HistorySize   int
	Auth          string
}

// DefaultTestServerOptions returns sensible defaults for testing.
func DefaultTestServerOptions() TestServerOptions {
	return TestServerOptions{
		IPs:           []string{"127.0.0.1"},
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 100,
		MaxConnsTotal: 1000,
		HistoryWindow: 5 * time.Minute,
		HistorySize:   100,
	}
}

// newTestConfig creates a test configuration.
func newTestConfig(opts TestServerOptions) *config.Config {
	cfg := config.DefaultConfig()
	cfg.IPs = opts.IPs
	cfg.Port = 0
	cfg.MetricsPort = 0
	cfg.Timeout = opts.Timeout
	cfg.IdleTimeout = opts.IdleTimeout
	cfg.MaxConnsPerIP = opts.MaxConnsPerIP
	cfg.MaxConnsTotal = opts.MaxConnsTotal
	cfg.HistoryWindow = opts.HistoryWindow
	cfg.HistorySize = opts.HistorySize
	cfg.LogLevel = "error"
	cfg.LogFormat = "json"
	cfg.Auth = opts.Auth
	return cfg
}

// newTestServerWithOptions creates a test proxy server with custom options.
func newTestServerWithOptions(t *testing.T, opts TestServerOptions) *Server {
	t.Helper()

	cfg := newTestConfig(opts)
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

// newTestServerWithIPs creates a test server with multiple IPs.
func newTestServerWithIPs(t *testing.T, ips []string) *Server {
	t.Helper()
	opts := DefaultTestServerOptions()
	opts.IPs = ips
	return newTestServerWithOptions(t, opts)
}

// newTestServerWithLimits creates a test server with custom connection limits.
func newTestServerWithLimits(t *testing.T, maxPerIP, maxTotal int) *Server {
	t.Helper()
	opts := DefaultTestServerOptions()
	opts.MaxConnsPerIP = maxPerIP
	opts.MaxConnsTotal = maxTotal
	return newTestServerWithOptions(t, opts)
}

// newTestServerWithTimeouts creates a test server with custom timeouts.
func newTestServerWithTimeouts(t *testing.T, timeout, idleTimeout time.Duration) *Server {
	t.Helper()
	opts := DefaultTestServerOptions()
	opts.Timeout = timeout
	opts.IdleTimeout = idleTimeout
	return newTestServerWithOptions(t, opts)
}

// newTestBackend creates a simple test HTTP backend.
func newTestBackend(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
}

// newTestBackendWithHandler creates a test backend with a custom handler.
func newTestBackendWithHandler(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// newTestBackendEcho creates a backend that echoes request info.
func newTestBackendEcho(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Host", r.Host)
		w.Header().Set("X-Request-Method", r.Method)
		w.Header().Set("X-Request-URI", r.RequestURI)
		w.WriteHeader(http.StatusOK)
	}))
}

// newTestBackendSlow creates a backend that responds slowly.
func newTestBackendSlow(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
}

// newTestBackendError creates a backend that returns errors.
func newTestBackendError(t *testing.T, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))
}

// newTestRequest creates a test HTTP request.
func newTestRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, url, nil)
	return req
}

// newTestRequestWithHost creates a test request with a specific host.
func newTestRequestWithHost(t *testing.T, method, url, host string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, url, nil)
	req.Host = host
	return req
}

// assertStatusCode checks that the response has the expected status code.
func assertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("expected status %d, got %d", expected, rr.Code)
	}
}

// assertHeader checks that a response header has the expected value.
func assertHeader(t *testing.T, rr *httptest.ResponseRecorder, key, expected string) {
	t.Helper()
	if got := rr.Header().Get(key); got != expected {
		t.Errorf("expected header %s=%s, got %s", key, expected, got)
	}
}

// assertNoError checks that an error is nil.
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// assertError checks that an error is not nil.
func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error, got nil")
	}
}
