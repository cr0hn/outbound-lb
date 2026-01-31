package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// TestProxy_10KConnections tests handling of 10,000 concurrent connections.
func TestProxy_10KConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10K connections test in short mode")
	}

	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 15000,
		MaxConnsTotal: 15000,
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

	// Create a fast backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	numConnections := 10000
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	startMem := getMemStats()
	startGoroutines := runtime.NumGoroutine()
	start := time.Now()

	// Launch 10K concurrent requests
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				successCount.Add(1)
			} else {
				errorCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	endMem := getMemStats()
	endGoroutines := runtime.NumGoroutine()

	t.Logf("=== 10K Connections Test Results ===")
	t.Logf("Total requests: %d", numConnections)
	t.Logf("Successful: %d", successCount.Load())
	t.Logf("Errors: %d", errorCount.Load())
	t.Logf("Duration: %v", elapsed)
	t.Logf("Requests/sec: %.2f", float64(numConnections)/elapsed.Seconds())
	t.Logf("Memory before: %d MB", startMem.Alloc/1024/1024)
	t.Logf("Memory after: %d MB", endMem.Alloc/1024/1024)
	t.Logf("Memory delta: %d MB", (endMem.Alloc-startMem.Alloc)/1024/1024)
	t.Logf("Goroutines before: %d", startGoroutines)
	t.Logf("Goroutines after: %d", endGoroutines)

	// Allow some goroutine variance (GC, etc.)
	goroutineDelta := endGoroutines - startGoroutines
	if goroutineDelta > 100 {
		t.Errorf("possible goroutine leak: %d new goroutines", goroutineDelta)
	}

	// Most requests should succeed
	successRate := float64(successCount.Load()) / float64(numConnections) * 100
	if successRate < 90 {
		t.Errorf("success rate too low: %.2f%%", successRate)
	}
}

// TestProxy_HighConcurrency tests with varying levels of concurrency.
func TestProxy_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping high concurrency test in short mode")
	}

	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 50000,
		MaxConnsTotal: 50000,
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

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	concurrencyLevels := []int{100, 500, 1000, 2000}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			var wg sync.WaitGroup
			var successCount atomic.Int64

			start := time.Now()

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)

					if rr.Code == http.StatusOK {
						successCount.Add(1)
					}
				}()
			}

			wg.Wait()
			elapsed := time.Since(start)

			t.Logf("Concurrency %d: %d successes in %v (%.2f req/s)",
				concurrency, successCount.Load(), elapsed,
				float64(concurrency)/elapsed.Seconds())
		})
	}
}

// TestProxy_SustainedLoad tests behavior under sustained load.
func TestProxy_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sustained load test in short mode")
	}

	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 1000,
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

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	duration := 5 * time.Second
	concurrency := 100
	var totalRequests atomic.Int64
	var successCount atomic.Int64

	ctx := make(chan struct{})
	var wg sync.WaitGroup

	// Track memory over time
	startMem := getMemStats()

	// Launch workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx:
					return
				default:
					req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)

					totalRequests.Add(1)
					if rr.Code == http.StatusOK {
						successCount.Add(1)
					}
				}
			}
		}()
	}

	// Run for duration
	time.Sleep(duration)
	close(ctx)
	wg.Wait()

	endMem := getMemStats()

	t.Logf("=== Sustained Load Test Results ===")
	t.Logf("Duration: %v", duration)
	t.Logf("Concurrency: %d", concurrency)
	t.Logf("Total requests: %d", totalRequests.Load())
	t.Logf("Successful: %d", successCount.Load())
	t.Logf("Requests/sec: %.2f", float64(totalRequests.Load())/duration.Seconds())
	t.Logf("Memory before: %d MB", startMem.Alloc/1024/1024)
	t.Logf("Memory after: %d MB", endMem.Alloc/1024/1024)

	// Success rate should be high
	successRate := float64(successCount.Load()) / float64(totalRequests.Load()) * 100
	if successRate < 95 {
		t.Errorf("success rate too low under sustained load: %.2f%%", successRate)
	}
}

// TestProxy_MultipleIPs tests load distribution across multiple IPs.
func TestProxy_MultipleIPs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multiple IPs test in short mode")
	}

	ips := []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.0.4"}

	cfg := &config.Config{
		IPs:           ips,
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

	statsCollector := metrics.NewStatsCollector(cfg.IPs)
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

	server := NewServer(cfg, bal, lim, statsCollector)

	numRequests := 1000
	hosts := []string{"host1.com", "host2.com", "host3.com"}

	for i := 0; i < numRequests; i++ {
		host := hosts[i%len(hosts)]
		ip, err := server.balancer.Select(host)
		if err != nil {
			t.Fatalf("select failed: %v", err)
		}
		server.balancer.Record(host, ip)
	}

	// Check distribution
	stats := bal.GetStats()
	t.Logf("Total hosts: %d", stats.TotalHosts)
	t.Logf("Total entries: %d", stats.TotalEntries)
	for ip, count := range stats.EntriesPerIP {
		t.Logf("IP %s: %d selections (%.1f%%)", ip, count, float64(count)/float64(numRequests)*100)
	}

	// Distribution should be reasonably even
	expectedPerIP := numRequests / len(ips)
	for ip, count := range stats.EntriesPerIP {
		deviation := float64(count-expectedPerIP) / float64(expectedPerIP) * 100
		if deviation > 50 || deviation < -50 {
			t.Logf("IP %s has significant deviation: %.1f%%", ip, deviation)
		}
	}
}

func getMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}
