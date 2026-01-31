// Package proxy provides the HTTP/HTTPS proxy server.
package proxy

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cr0hn/outbound-lb/internal/balancer"
	"github.com/cr0hn/outbound-lb/internal/config"
	"github.com/cr0hn/outbound-lb/internal/limiter"
	"github.com/cr0hn/outbound-lb/internal/logger"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// Server is the HTTP/HTTPS proxy server.
type Server struct {
	cfg            *config.Config
	httpServer     *http.Server
	balancer       balancer.Balancer
	limiter        *limiter.Limiter
	transportPool  *TransportPool
	stats          *metrics.StatsCollector
	connectHandler *ConnectHandler
}

// NewServer creates a new proxy server.
func NewServer(cfg *config.Config, bal balancer.Balancer, lim *limiter.Limiter, stats *metrics.StatsCollector) *Server {
	s := &Server{
		cfg:           cfg,
		balancer:      bal,
		limiter:       lim,
		transportPool: NewTransportPool(cfg.IPs, cfg.Timeout),
		stats:         stats,
	}

	// Create handlers
	handler := NewHandler(s)
	s.connectHandler = NewConnectHandler(s)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Start starts the proxy server.
func (s *Server) Start() error {
	logger.Info("starting proxy server",
		"port", s.cfg.Port,
		"ips", s.cfg.IPs,
		"auth_enabled", s.cfg.Auth != "",
	)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("shutting down proxy server")
	s.transportPool.Close()
	return s.httpServer.Shutdown(ctx)
}

// authenticate checks if the request is authenticated.
func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) bool {
	// No auth configured
	if s.cfg.Auth == "" {
		return true
	}

	username, password, ok := s.cfg.GetAuthCredentials()
	if !ok {
		return true // Invalid config, skip auth
	}

	// Get Proxy-Authorization header
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		s.sendProxyAuthRequired(w)
		metrics.AuthFailures.Inc()
		return false
	}

	// Parse Basic auth
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		s.sendProxyAuthRequired(w)
		metrics.AuthFailures.Inc()
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		s.sendProxyAuthRequired(w)
		metrics.AuthFailures.Inc()
		return false
	}

	credentials := string(decoded)
	colonIdx := strings.Index(credentials, ":")
	if colonIdx < 0 {
		s.sendProxyAuthRequired(w)
		metrics.AuthFailures.Inc()
		return false
	}

	reqUser := credentials[:colonIdx]
	reqPass := credentials[colonIdx+1:]

	// Use constant-time comparison to prevent timing attacks
	userMatch := subtle.ConstantTimeCompare([]byte(reqUser), []byte(username)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(reqPass), []byte(password)) == 1
	if !userMatch || !passMatch {
		logger.Warn("authentication failed", "user", reqUser, "remote", r.RemoteAddr)
		s.sendProxyAuthRequired(w)
		metrics.AuthFailures.Inc()
		return false
	}

	return true
}

// sendProxyAuthRequired sends a 407 Proxy Authentication Required response.
func (s *Server) sendProxyAuthRequired(w http.ResponseWriter) {
	w.Header().Set("Proxy-Authenticate", `Basic realm="Proxy"`)
	http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
}

// selectIP selects an outbound IP for the given host.
func (s *Server) selectIP(host string) (string, error) {
	return s.balancer.Select(host)
}

// ConnectionContext holds information about an acquired connection.
type ConnectionContext struct {
	IP        string
	Host      string
	RequestID string
	release   func()
}

// Release releases the connection resources. Must be called when done.
func (c *ConnectionContext) Release() {
	if c.release != nil {
		c.release()
	}
}

// AcquireConnection selects an IP and acquires a connection slot.
// Returns a ConnectionContext that must be released when done.
// Returns an error if no IPs are available or connection limit is reached.
func (s *Server) AcquireConnection(host, requestID string) (*ConnectionContext, error) {
	// Select outbound IP
	logger.Trace("connection_acquire_start", "request_id", requestID, "host", host)
	ip, err := s.selectIP(host)
	if err != nil {
		logger.Trace("connection_ip_selection_failed", "request_id", requestID, "host", host, "error", err)
		return nil, err
	}
	logger.Trace("connection_ip_selected", "request_id", requestID, "host", host, "ip", ip)

	// Acquire connection slot
	if err := s.limiter.Acquire(ip); err != nil {
		logger.Trace("connection_acquire_failed", "request_id", requestID, "ip", ip, "error", err)
		return nil, err
	}
	logger.Trace("connection_acquired", "request_id", requestID, "ip", ip)

	// Update metrics
	s.stats.IncActiveConnections()
	s.stats.IncConnectionsForIP(ip)

	// Record selection
	s.balancer.Record(host, ip)
	s.stats.IncSelectionsForIP(ip, host)
	logger.LogBalancerSelection(host, ip, len(s.cfg.IPs))

	return &ConnectionContext{
		IP:        ip,
		Host:      host,
		RequestID: requestID,
		release: func() {
			s.limiter.Release(ip)
			s.stats.DecActiveConnections()
			s.stats.DecConnectionsForIP(ip)
		},
	}, nil
}

// WaitForConnections waits for active connections to complete.
func (s *Server) WaitForConnections(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.limiter.GetTotalCount() == 0 {
				logger.Info("all connections closed")
				return
			}
			if time.Now().After(deadline) {
				logger.Warn("timeout waiting for connections",
					"active", s.limiter.GetTotalCount(),
				)
				return
			}
		}
	}
}
