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

func newBenchServer(b *testing.B) *Server {
	b.Helper()
	cfg := &config.Config{
		IPs:           []string{"127.0.0.1"},
		Port:          0,
		MetricsPort:   0,
		Timeout:       30 * time.Second,
		IdleTimeout:   60 * time.Second,
		MaxConnsPerIP: 100000,
		MaxConnsTotal: 1000000,
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

func BenchmarkHandler_ServeHTTP(b *testing.B) {
	server := newBenchServer(b)
	handler := NewHandler(server)

	// Create a mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkHandler_ServeHTTP_Parallel(b *testing.B) {
	server := newBenchServer(b)
	handler := NewHandler(server)

	// Create a mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, backend.URL, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

func BenchmarkGenerateRequestID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateRequestID()
	}
}

func BenchmarkGenerateRequestID_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GenerateRequestID()
		}
	})
}

func BenchmarkHandler_createOutgoingRequest(b *testing.B) {
	server := newBenchServer(b)
	handler := NewHandler(server)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	req.Header.Set("X-Custom-Header", "value")
	req.Header.Set("Connection", "keep-alive")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.createOutgoingRequest(req)
	}
}

func BenchmarkHandler_copyHeaders(b *testing.B) {
	server := newBenchServer(b)
	handler := NewHandler(server)

	src := http.Header{}
	for i := 0; i < 20; i++ {
		src.Add("X-Header-"+string(rune('A'+i)), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst := http.Header{}
		handler.copyHeaders(dst, src)
	}
}

func BenchmarkHandler_removeHopByHopHeaders(b *testing.B) {
	server := newBenchServer(b)
	handler := NewHandler(server)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		header := http.Header{}
		header.Set("Connection", "keep-alive")
		header.Set("Keep-Alive", "timeout=5")
		header.Set("Proxy-Authorization", "Basic xxx")
		header.Set("X-Custom", "value")
		handler.removeHopByHopHeaders(header)
	}
}

func BenchmarkHandler_getClientIP(b *testing.B) {
	server := newBenchServer(b)
	handler := NewHandler(server)

	tests := []struct {
		name       string
		remoteAddr string
	}{
		{"IPv4", "192.168.1.1:12345"},
		{"IPv6", "[::1]:12345"},
		{"IPv6Long", "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:12345"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			req.RemoteAddr = tt.remoteAddr

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = handler.getClientIP(req)
			}
		})
	}
}

func BenchmarkHandler_authenticate_NoAuth(b *testing.B) {
	server := newBenchServer(b)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		_ = server.authenticate(rr, req)
	}
}

func BenchmarkHandler_isHopByHop(b *testing.B) {
	headers := []string{
		"Connection",
		"Keep-Alive",
		"X-Custom-Header",
		"Content-Type",
		"Proxy-Authorization",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, h := range headers {
			_ = isHopByHop(h)
		}
	}
}
