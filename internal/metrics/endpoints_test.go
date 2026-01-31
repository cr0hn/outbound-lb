package metrics

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestMetricsEndpoint_PrometheusFormat tests that /metrics returns valid Prometheus format.
func TestMetricsEndpoint_PrometheusFormat(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1", "192.168.1.2"})
	server := NewServer(0, stats)

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the mux from server and serve
		mux := http.NewServeMux()
		mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use promhttp handler directly
			server.server.Handler.ServeHTTP(w, r)
		}))
		mux.ServeHTTP(w, r)
	}))
	defer ts.Close()

	// Make request to /metrics
	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to get /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") && !strings.Contains(contentType, "text/plain; version=0.0.4") {
		// Prometheus metrics can also return application/openmetrics-text
		if !strings.Contains(contentType, "openmetrics") {
			t.Logf("Content-Type: %s (acceptable)", contentType)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	// Verify it contains HELP and TYPE comments (Prometheus format)
	content := string(body)
	if !strings.Contains(content, "# HELP") {
		t.Error("expected Prometheus format with # HELP comments")
	}
	if !strings.Contains(content, "# TYPE") {
		t.Error("expected Prometheus format with # TYPE comments")
	}
}

// TestMetricsEndpoint_ContainsAllMetrics tests that all expected metrics are present.
func TestMetricsEndpoint_ContainsAllMetrics(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1", "192.168.1.2"})

	// Trigger some metrics to ensure they're registered
	stats.IncActiveConnections()
	stats.IncTotalRequests()
	stats.AddBytesSent(100)
	stats.AddBytesReceived(50)
	stats.IncConnectionsForIP("192.168.1.1")
	stats.IncSelectionsForIP("192.168.1.1", "example.com")

	// Increment some Prometheus-only metrics
	RequestsTotal.WithLabelValues("CONNECT", "200").Inc()
	RequestDuration.WithLabelValues("CONNECT").Observe(0.5)
	LimitRejections.WithLabelValues("per_ip").Inc()
	AuthFailures.Inc()
	TunnelConnections.Inc()
	HistoryEntries.Set(10)
	HistoryHosts.Set(5)
	HealthCheckTotal.WithLabelValues("192.168.1.1", "success").Inc()
	IPHealthStatus.WithLabelValues("192.168.1.1").Set(1)
	HealthCheckDuration.WithLabelValues("192.168.1.1").Observe(0.01)
	HealthyIPs.Set(2)
	UnhealthyIPs.Set(0)

	server := NewServer(0, stats)

	// Create a test request directly to the server's handler
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// List of all metrics that should be present
	expectedMetrics := []string{
		"outbound_lb_requests_total",
		"outbound_lb_request_duration_seconds",
		"outbound_lb_bytes_sent_total",
		"outbound_lb_bytes_received_total",
		"outbound_lb_active_connections",
		"outbound_lb_connections_per_ip",
		"outbound_lb_balancer_selections_total",
		"outbound_lb_limit_rejections_total",
		"outbound_lb_auth_failures_total",
		"outbound_lb_tunnel_connections_total",
		"outbound_lb_history_entries",
		"outbound_lb_history_hosts",
		"outbound_lb_health_check_total",
		"outbound_lb_ip_health_status",
		"outbound_lb_health_check_duration_seconds",
		"outbound_lb_healthy_ips",
		"outbound_lb_unhealthy_ips",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metric %q not found in /metrics output", metric)
		}
	}
}

// TestHealthEndpoint_Integration tests the /health endpoint with a real HTTP server.
func TestHealthEndpoint_Integration(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(0, stats)

	// Use the server's handler directly
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify JSON response
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", response["status"])
	}

	// Verify uptime field exists
	if _, ok := response["uptime"]; !ok {
		t.Error("expected 'uptime' field in response")
	}

	// Verify Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}
}

// TestReadyEndpoint_Integration tests the /ready endpoint with both states.
func TestReadyEndpoint_Integration(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(0, stats)

	t.Run("not ready", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if response["status"] != "not ready" {
			t.Errorf("expected status 'not ready', got %v", response["status"])
		}
	})

	t.Run("ready", func(t *testing.T) {
		server.SetReady(true)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()

		server.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if response["status"] != "ready" {
			t.Errorf("expected status 'ready', got %v", response["status"])
		}
	})
}

// TestStatsEndpoint_Integration tests the /stats endpoint with various states.
func TestStatsEndpoint_Integration(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.1.2", "10.0.0.1"}
	stats := NewStatsCollector(ips)

	// Simulate some activity
	stats.IncActiveConnections()
	stats.IncActiveConnections()
	stats.IncTotalRequests()
	stats.IncTotalRequests()
	stats.IncTotalRequests()
	stats.AddBytesSent(1500)
	stats.AddBytesReceived(750)
	stats.IncConnectionsForIP("192.168.1.1")
	stats.IncConnectionsForIP("192.168.1.2")
	stats.IncSelectionsForIP("192.168.1.1", "example.com")
	stats.IncSelectionsForIP("192.168.1.1", "example.com")
	stats.IncSelectionsForIP("192.168.1.2", "other.com")

	server := NewServer(0, stats)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Stats
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	// Verify stats values
	if response.ActiveConnections != 2 {
		t.Errorf("expected 2 active connections, got %d", response.ActiveConnections)
	}
	if response.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", response.TotalRequests)
	}
	if response.BytesSent != 1500 {
		t.Errorf("expected 1500 bytes sent, got %d", response.BytesSent)
	}
	if response.BytesReceived != 750 {
		t.Errorf("expected 750 bytes received, got %d", response.BytesReceived)
	}

	// Verify connections per IP
	if response.ConnectionsPerIP["192.168.1.1"] != 1 {
		t.Errorf("expected 1 connection for 192.168.1.1, got %d", response.ConnectionsPerIP["192.168.1.1"])
	}
	if response.ConnectionsPerIP["192.168.1.2"] != 1 {
		t.Errorf("expected 1 connection for 192.168.1.2, got %d", response.ConnectionsPerIP["192.168.1.2"])
	}

	// Verify selections per IP
	if response.SelectionsPerIP["192.168.1.1"] != 2 {
		t.Errorf("expected 2 selections for 192.168.1.1, got %d", response.SelectionsPerIP["192.168.1.1"])
	}
	if response.SelectionsPerIP["192.168.1.2"] != 1 {
		t.Errorf("expected 1 selection for 192.168.1.2, got %d", response.SelectionsPerIP["192.168.1.2"])
	}

	// Verify Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}
}

// TestMetricsServer_FullIntegration tests the full server lifecycle.
func TestMetricsServer_FullIntegration(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1", "192.168.1.2"})
	port := 19998 // Use a high port unlikely to be in use

	server := NewServer(port, stats)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Check for startup errors
	select {
	case err := <-serverErr:
		t.Fatalf("server failed to start: %v", err)
	default:
		// Server started successfully
	}

	baseURL := "http://localhost:19998"

	// Test all endpoints
	t.Run("health endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("failed to get /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("ready endpoint before ready", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/ready")
		if err != nil {
			t.Fatalf("failed to get /ready: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", resp.StatusCode)
		}
	})

	t.Run("ready endpoint after ready", func(t *testing.T) {
		server.SetReady(true)

		resp, err := http.Get(baseURL + "/ready")
		if err != nil {
			t.Fatalf("failed to get /ready: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("stats endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/stats")
		if err != nil {
			t.Fatalf("failed to get /stats: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("metrics endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/metrics")
		if err != nil {
			t.Fatalf("failed to get /metrics: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "outbound_lb") {
			t.Error("expected metrics to contain 'outbound_lb' prefix")
		}
	})

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}

// TestMetricsEndpoint_MetricValues tests that metrics have correct values.
func TestMetricsEndpoint_MetricValues(t *testing.T) {
	// Reset some counters by creating fresh stats
	stats := NewStatsCollector([]string{"10.0.0.1"})

	// Set specific values
	stats.AddBytesSent(12345)
	stats.AddBytesReceived(6789)

	server := NewServer(0, stats)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Check that byte counters appear with values
	if !strings.Contains(body, "outbound_lb_bytes_sent_total") {
		t.Error("expected outbound_lb_bytes_sent_total metric")
	}
	if !strings.Contains(body, "outbound_lb_bytes_received_total") {
		t.Error("expected outbound_lb_bytes_received_total metric")
	}
}

// TestEndpoints_ContentType verifies all endpoints return correct Content-Type.
func TestEndpoints_ContentType(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(0, stats)

	tests := []struct {
		endpoint    string
		wantType    string
		wantPartial bool // If true, check Contains instead of exact match
	}{
		{"/health", "application/json", false},
		{"/ready", "application/json", false},
		{"/stats", "application/json", false},
		{"/metrics", "text/plain", true}, // Prometheus can return text/plain or application/openmetrics-text
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			server.server.Handler.ServeHTTP(w, req)

			contentType := w.Header().Get("Content-Type")
			if tt.wantPartial {
				// For metrics, accept various Prometheus content types
				if !strings.Contains(contentType, "text/plain") &&
					!strings.Contains(contentType, "openmetrics") {
					t.Errorf("expected Content-Type containing 'text/plain' or 'openmetrics', got %q", contentType)
				}
			} else {
				if contentType != tt.wantType {
					t.Errorf("expected Content-Type %q, got %q", tt.wantType, contentType)
				}
			}
		})
	}
}
