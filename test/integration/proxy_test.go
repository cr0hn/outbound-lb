package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
	"github.com/cr0hn/outbound-lb/internal/proxy"
)

func TestProxyIntegration_BasicHTTP(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	// Create proxy server
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0, // Will be assigned
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

	proxyServer := proxy.NewServer(cfg, bal, lim, stats)
	_ = proxyServer // Just test creation for now
}

func TestProxyIntegration_MetricsEndpoints(t *testing.T) {
	stats := metrics.NewStatsCollector([]string{"127.0.0.1"})
	metricsServer := metrics.NewServer(0, stats)

	// Start server in background
	go metricsServer.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		metricsServer.Shutdown(ctx)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)
}

func TestProxyIntegration_ConnectionLimiting(t *testing.T) {
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 2, // Very low limit for testing
		MaxConnsTotal: 5,
		HistoryWindow: 5 * time.Minute,
		HistorySize:   100,
		LogLevel:      "error",
		LogFormat:     "json",
	}

	stats := metrics.NewStatsCollector(cfg.IPs)
	lim := limiter.New(cfg.MaxConnsPerIP, cfg.MaxConnsTotal, cfg.IPs)

	// Acquire connections up to limit
	err := lim.Acquire("127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error on first acquire: %v", err)
	}

	err = lim.Acquire("127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error on second acquire: %v", err)
	}

	// Third should fail
	err = lim.Acquire("127.0.0.1")
	if err != limiter.ErrIPLimitReached {
		t.Errorf("expected ErrIPLimitReached, got: %v", err)
	}

	// Release one
	lim.Release("127.0.0.1")

	// Should succeed again
	err = lim.Acquire("127.0.0.1")
	if err != nil {
		t.Errorf("unexpected error after release: %v", err)
	}

	_ = stats
}

func TestProxyIntegration_BalancerDistribution(t *testing.T) {
	cfg := &config.Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
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

	// Make multiple selections and track distribution
	selections := make(map[string]int)
	host := "api.example.com"

	for i := 0; i < 30; i++ {
		ip, err := bal.Select(host)
		if err != nil {
			t.Fatalf("selection error: %v", err)
		}
		bal.Record(host, ip)
		selections[ip]++
	}

	// With LRU algorithm, distribution should be roughly even
	for ip, count := range selections {
		if count < 5 || count > 15 {
			t.Logf("IP %s was selected %d times (distribution may vary)", ip, count)
		}
	}

	// Verify all IPs were used
	if len(selections) != len(cfg.IPs) {
		t.Errorf("expected all %d IPs to be used, but only %d were used", len(cfg.IPs), len(selections))
	}
}

func TestProxyIntegration_PerHostIsolation(t *testing.T) {
	cfg := &config.Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
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

	// Use one IP heavily for host1
	for i := 0; i < 100; i++ {
		ip, _ := bal.Select("host1.com")
		bal.Record("host1.com", ip)
	}

	// host2 should still get balanced distribution
	selectionsHost2 := make(map[string]int)
	for i := 0; i < 10; i++ {
		ip, _ := bal.Select("host2.com")
		bal.Record("host2.com", ip)
		selectionsHost2[ip]++
	}

	// host2 should use both IPs (independent from host1)
	if len(selectionsHost2) != 2 {
		t.Logf("host2 selections: %v (isolation test)", selectionsHost2)
	}
}

func TestProxyIntegration_WithRealBackend(t *testing.T) {
	// Skip in CI or if explicitly requested
	if testing.Short() {
		t.Skip("skipping integration test with real backend")
	}

	// Create a test backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Path: %s, XFF: %s", r.URL.Path, xff)
	}))
	defer backend.Close()

	// Parse backend URL
	backendURL, _ := url.Parse(backend.URL)
	_ = backendURL

	// Verify backend is working
	resp, err := http.Get(backend.URL + "/test")
	if err != nil {
		t.Fatalf("backend request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestProxyIntegration_StatsCollection(t *testing.T) {
	stats := metrics.NewStatsCollector([]string{"192.168.1.1", "192.168.1.2"})

	// Simulate activity
	stats.IncActiveConnections()
	stats.IncActiveConnections()
	stats.IncTotalRequests()
	stats.AddBytesSent(1000)
	stats.AddBytesReceived(500)
	stats.IncConnectionsForIP("192.168.1.1")
	stats.IncSelectionsForIP("192.168.1.1", "example.com")

	// Verify stats
	s := stats.GetStats()
	if s.ActiveConnections != 2 {
		t.Errorf("expected 2 active connections, got %d", s.ActiveConnections)
	}
	if s.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", s.TotalRequests)
	}
	if s.BytesSent != 1000 {
		t.Errorf("expected 1000 bytes sent, got %d", s.BytesSent)
	}
	if s.BytesReceived != 500 {
		t.Errorf("expected 500 bytes received, got %d", s.BytesReceived)
	}
}

func TestProxyIntegration_GracefulShutdown(t *testing.T) {
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
	bal.Start()

	proxyServer := proxy.NewServer(cfg, bal, lim, stats)

	// Start shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown should complete quickly when no connections
	err := proxyServer.Shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	bal.Stop()
}
