package proxy

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

func newTestServerWithAuth(t *testing.T, auth string) *Server {
	t.Helper()
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Auth:          auth,
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

func TestServer_NewServer(t *testing.T) {
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          3128,
		MetricsPort:   9090,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 100,
		MaxConnsTotal: 1000,
		HistoryWindow: 5 * time.Minute,
		HistorySize:   100,
		LogLevel:      "info",
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

	server := NewServer(cfg, bal, lim, stats)

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.cfg != cfg {
		t.Error("expected config to be set")
	}
	if server.balancer == nil {
		t.Error("expected balancer to be set")
	}
	if server.limiter == nil {
		t.Error("expected limiter to be set")
	}
	if server.transportPool == nil {
		t.Error("expected transport pool to be set")
	}
	if server.stats == nil {
		t.Error("expected stats to be set")
	}
}

func TestServer_Authenticate_NoAuth(t *testing.T) {
	server := newTestServerWithAuth(t, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if !result {
		t.Error("expected authentication to pass when no auth configured")
	}
}

func TestServer_Authenticate_MissingHeader(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if result {
		t.Error("expected authentication to fail when header missing")
	}
	if w.Code != http.StatusProxyAuthRequired {
		t.Errorf("expected status 407, got %d", w.Code)
	}
}

func TestServer_Authenticate_InvalidScheme(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Proxy-Authorization", "Bearer token")
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if result {
		t.Error("expected authentication to fail with invalid scheme")
	}
}

func TestServer_Authenticate_InvalidBase64(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Proxy-Authorization", "Basic not-valid-base64!!!")
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if result {
		t.Error("expected authentication to fail with invalid base64")
	}
}

func TestServer_Authenticate_MissingColon(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	encoded := base64.StdEncoding.EncodeToString([]byte("userpass"))
	req.Header.Set("Proxy-Authorization", "Basic "+encoded)
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if result {
		t.Error("expected authentication to fail without colon separator")
	}
}

func TestServer_Authenticate_WrongCredentials(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	encoded := base64.StdEncoding.EncodeToString([]byte("wrong:credentials"))
	req.Header.Set("Proxy-Authorization", "Basic "+encoded)
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if result {
		t.Error("expected authentication to fail with wrong credentials")
	}
}

func TestServer_Authenticate_ValidCredentials(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	encoded := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	req.Header.Set("Proxy-Authorization", "Basic "+encoded)
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if !result {
		t.Error("expected authentication to pass with valid credentials")
	}
}

func TestServer_Authenticate_PasswordWithColon(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass:word")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	encoded := base64.StdEncoding.EncodeToString([]byte("user:pass:word"))
	req.Header.Set("Proxy-Authorization", "Basic "+encoded)
	w := httptest.NewRecorder()

	result := server.authenticate(w, req)
	if !result {
		t.Error("expected authentication to pass with password containing colon")
	}
}

func TestServer_SelectIP(t *testing.T) {
	server := newTestServerWithAuth(t, "")

	ip, err := server.selectIP("example.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", ip)
	}
}

func TestServer_SendProxyAuthRequired(t *testing.T) {
	server := newTestServerWithAuth(t, "user:pass")

	w := httptest.NewRecorder()
	server.sendProxyAuthRequired(w)

	if w.Code != http.StatusProxyAuthRequired {
		t.Errorf("expected status 407, got %d", w.Code)
	}

	authHeader := w.Header().Get("Proxy-Authenticate")
	if authHeader != `Basic realm="Proxy"` {
		t.Errorf("expected Proxy-Authenticate header, got %s", authHeader)
	}
}

func TestServer_WaitForConnections_NoConnections(t *testing.T) {
	server := newTestServerWithAuth(t, "")

	// Should return quickly when no connections
	start := time.Now()
	server.WaitForConnections(1 * time.Second)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("expected quick return, took %v", elapsed)
	}
}
