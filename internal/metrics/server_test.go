package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(9090, stats)

	if server == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestServer_HealthHandler(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(9090, stats)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Access handler directly through test
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.healthHandler)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", response["status"])
	}
}

func TestServer_ReadyHandler_NotReady(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(9090, stats)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", server.readyHandler)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestServer_ReadyHandler_Ready(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(9090, stats)
	server.SetReady(true)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/ready", server.readyHandler)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestServer_StatsHandler(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1", "192.168.1.2"})
	stats.IncActiveConnections()
	stats.IncTotalRequests()
	stats.AddBytesSent(1000)

	server := NewServer(9090, stats)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/stats", server.statsHandler)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response Stats
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.ActiveConnections != 1 {
		t.Errorf("expected 1 active connection, got %d", response.ActiveConnections)
	}
	if response.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", response.TotalRequests)
	}
	if response.BytesSent != 1000 {
		t.Errorf("expected 1000 bytes sent, got %d", response.BytesSent)
	}
}

func TestServer_SetReady(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(9090, stats)

	if server.ready.Load() {
		t.Error("expected ready to be false initially")
	}

	server.SetReady(true)
	if !server.ready.Load() {
		t.Error("expected ready to be true after SetReady(true)")
	}

	server.SetReady(false)
	if server.ready.Load() {
		t.Error("expected ready to be false after SetReady(false)")
	}
}

func TestServer_Shutdown(t *testing.T) {
	stats := NewStatsCollector([]string{"192.168.1.1"})
	server := NewServer(19999, stats)

	// Start server in background
	go func() {
		server.Start()
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}
