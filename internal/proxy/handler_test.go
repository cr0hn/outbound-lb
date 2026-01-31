package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

func newTestServer(t *testing.T) *Server {
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

func TestHandler_isHopByHop(t *testing.T) {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Proxy-Connection",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, h := range hopByHopHeaders {
		if !isHopByHop(h) {
			t.Errorf("expected %s to be hop-by-hop header", h)
		}
	}

	if isHopByHop("Content-Type") {
		t.Error("Content-Type should not be hop-by-hop")
	}
	if isHopByHop("X-Custom-Header") {
		t.Error("X-Custom-Header should not be hop-by-hop")
	}
}

func TestHandler_copyHeaders(t *testing.T) {
	server := newTestServer(t)
	handler := NewHandler(server)

	src := http.Header{}
	src.Set("Content-Type", "application/json")
	src.Set("X-Custom", "value")
	src.Set("Connection", "keep-alive") // hop-by-hop, should be skipped

	dst := http.Header{}
	handler.copyHeaders(dst, src)

	if dst.Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type to be copied")
	}
	if dst.Get("X-Custom") != "value" {
		t.Error("expected X-Custom to be copied")
	}
	if dst.Get("Connection") != "" {
		t.Error("expected Connection (hop-by-hop) to NOT be copied")
	}
}

func TestHandler_removeHopByHopHeaders(t *testing.T) {
	server := newTestServer(t)
	handler := NewHandler(server)

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Connection", "keep-alive")
	headers.Set("Proxy-Authorization", "Basic abc")
	headers.Set("Keep-Alive", "timeout=5")

	handler.removeHopByHopHeaders(headers)

	if headers.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should not be removed")
	}
	if headers.Get("Connection") != "" {
		t.Error("Connection should be removed")
	}
	if headers.Get("Proxy-Authorization") != "" {
		t.Error("Proxy-Authorization should be removed")
	}
	if headers.Get("Keep-Alive") != "" {
		t.Error("Keep-Alive should be removed")
	}
}

func TestHandler_getClientIP(t *testing.T) {
	server := newTestServer(t)
	handler := NewHandler(server)

	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{"IPv4 with port", "192.168.1.1:12345", "192.168.1.1"},
		{"IPv4 with different port", "10.0.0.1:8080", "10.0.0.1"},
		{"IPv6 loopback with port", "[::1]:12345", "::1"},
		{"IPv6 full address with port", "[2001:db8::1]:8080", "2001:db8::1"},
		{"IPv6 link-local with port", "[fe80::1%eth0]:9000", "fe80::1%eth0"},
		{"IPv4 without port", "192.168.1.1", "192.168.1.1"},
		{"IPv6 without port (malformed)", "[::1]", "[::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr

			ip := handler.getClientIP(req)
			if ip != tt.expected {
				t.Errorf("getClientIP(%s) = %s, want %s", tt.remoteAddr, ip, tt.expected)
			}
		})
	}
}

func TestHandler_sendError(t *testing.T) {
	server := newTestServer(t)
	handler := NewHandler(server)

	w := httptest.NewRecorder()
	handler.sendError(w, http.StatusBadGateway, "Bad Gateway")

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "Bad Gateway") {
		t.Error("expected error message in body")
	}
}

func TestHandler_createOutgoingRequest(t *testing.T) {
	server := newTestServer(t)
	handler := NewHandler(server)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Custom", "value")
	req.Header.Set("Connection", "keep-alive")

	outReq := handler.createOutgoingRequest(req)

	if outReq.URL.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", outReq.URL.Host)
	}

	if outReq.Header.Get("X-Custom") != "value" {
		t.Error("expected X-Custom header to be preserved")
	}

	// Connection header should be removed (hop-by-hop)
	if outReq.Header.Get("Connection") != "" {
		t.Error("expected Connection header to be removed")
	}

	// X-Forwarded-For should be set
	xff := outReq.Header.Get("X-Forwarded-For")
	if xff != "192.168.1.100" {
		t.Errorf("expected X-Forwarded-For to be 192.168.1.100, got %s", xff)
	}
}

func TestHandler_createOutgoingRequest_ExistingXFF(t *testing.T) {
	server := newTestServer(t)
	handler := NewHandler(server)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	outReq := handler.createOutgoingRequest(req)

	xff := outReq.Header.Get("X-Forwarded-For")
	if xff != "10.0.0.1, 192.168.1.100" {
		t.Errorf("expected X-Forwarded-For to be '10.0.0.1, 192.168.1.100', got %s", xff)
	}
}
