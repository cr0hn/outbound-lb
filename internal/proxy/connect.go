// Package proxy provides the HTTP/HTTPS proxy server.
package proxy

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cr0hn/outbound-lb/internal/logger"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// ConnectHandler handles CONNECT tunnel requests.
type ConnectHandler struct {
	server *Server
}

// NewConnectHandler creates a new ConnectHandler.
func NewConnectHandler(server *Server) *ConnectHandler {
	return &ConnectHandler{server: server}
}

// ServeHTTP handles a CONNECT request.
func (h *ConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Get or generate request ID for tracing
	requestID := RequestIDFromContext(r.Context())
	if requestID == "" {
		requestID = GenerateRequestID()
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	logger.Trace("connect_request_received", "request_id", requestID, "host", host, "remote", r.RemoteAddr)

	// Select outbound IP
	logger.Trace("connect_ip_selection_start", "host", host)
	ip, err := h.server.selectIP(host)
	if err != nil {
		logger.Trace("connect_ip_selection_failed", "host", host, "error", err)
		http.Error(w, "No available outbound IPs", http.StatusServiceUnavailable)
		metrics.LimitRejections.WithLabelValues("total").Inc()
		return
	}
	logger.Trace("connect_ip_selected", "host", host, "ip", ip)

	// Acquire connection slot
	logger.Trace("connect_acquire_attempt", "ip", ip)
	if err := h.server.limiter.Acquire(ip); err != nil {
		logger.Trace("connect_acquire_failed", "ip", ip, "error", err)
		http.Error(w, "Connection limit reached", http.StatusServiceUnavailable)
		metrics.LimitRejections.WithLabelValues("per_ip").Inc()
		logger.LogConnectionLimit("per_ip", ip, int(h.server.limiter.GetIPCount(ip)), h.server.cfg.MaxConnsPerIP)
		return
	}
	logger.Trace("connect_acquired", "ip", ip)
	defer h.server.limiter.Release(ip)

	// Update metrics
	h.server.stats.IncActiveConnections()
	h.server.stats.IncConnectionsForIP(ip)
	defer func() {
		h.server.stats.DecActiveConnections()
		h.server.stats.DecConnectionsForIP(ip)
	}()

	// Record selection
	h.server.balancer.Record(host, ip)
	h.server.stats.IncSelectionsForIP(ip, host)
	logger.LogBalancerSelection(host, ip, len(h.server.cfg.IPs))

	metrics.TunnelConnections.Inc()

	// Create dialer for this IP
	dialer := NewDialer(ip, h.server.cfg.Timeout, h.server.cfg.IdleTimeout)

	// Connect to target
	logger.Trace("connect_dial_start", "host", host, "ip", ip)
	targetConn, err := dialer.Dial("tcp", host)
	if err != nil {
		logger.Trace("connect_dial_failed", "host", host, "ip", ip, "error", err)
		logger.LogError("connect_dial", err, "host", host, "ip", ip)
		http.Error(w, "Failed to connect to target", http.StatusBadGateway)
		metrics.RequestsTotal.WithLabelValues("CONNECT", "502").Inc()
		return
	}
	logger.Trace("connect_dial_success", "host", host, "ip", ip, "local", targetConn.LocalAddr(), "remote", targetConn.RemoteAddr())
	defer targetConn.Close()

	// Hijack client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		logger.LogError("connect_hijack", fmt.Errorf("hijacking not supported"), "host", host)
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		metrics.RequestsTotal.WithLabelValues("CONNECT", "500").Inc()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		logger.LogError("connect_hijack", err, "host", host)
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		metrics.RequestsTotal.WithLabelValues("CONNECT", "500").Inc()
		return
	}
	defer clientConn.Close()

	// Send 200 Connection Established
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		logger.LogError("connect_response", err, "host", host)
		return
	}

	// Bidirectional copy with idle timeout
	bytesIn, bytesOut := h.tunnel(clientConn, targetConn, h.server.cfg.IdleTimeout)

	// Log and record metrics
	duration := time.Since(start).Milliseconds()
	logger.LogRequest("CONNECT", host, r.RemoteAddr, ip, 200, duration, bytesIn, bytesOut)

	h.server.stats.IncTotalRequests()
	h.server.stats.AddBytesReceived(bytesIn)
	h.server.stats.AddBytesSent(bytesOut)

	metrics.RequestsTotal.WithLabelValues("CONNECT", "200").Inc()
	metrics.RequestDuration.WithLabelValues("CONNECT").Observe(time.Since(start).Seconds())
}

// tunnel performs bidirectional copy between two connections with idle timeout.
// The timeout is reset on each successful read/write operation.
func (h *ConnectHandler) tunnel(client, target net.Conn, idleTimeout time.Duration) (bytesIn, bytesOut int64) {
	var wg sync.WaitGroup
	var in, out atomic.Int64
	wg.Add(2)

	logger.Trace("tunnel_started", "client", client.RemoteAddr(), "target", target.RemoteAddr(), "idle_timeout", idleTimeout)

	// Set initial deadline
	deadline := time.Now().Add(idleTimeout)
	client.SetDeadline(deadline)
	target.SetDeadline(deadline)

	// Client -> Target
	go func() {
		defer wg.Done()
		n, err := copyWithIdleTimeout(target, client, idleTimeout)
		if err != nil && !errors.Is(err, net.ErrClosed) && !isTimeoutError(err) {
			logger.LogError("tunnel_client_to_target", err)
		}
		in.Store(n)
		logger.Trace("tunnel_transfer_complete", "direction", "client_to_target", "bytes", n)
		// Signal EOF to target
		if tc, ok := target.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	// Target -> Client
	go func() {
		defer wg.Done()
		n, err := copyWithIdleTimeout(client, target, idleTimeout)
		if err != nil && !errors.Is(err, net.ErrClosed) && !isTimeoutError(err) {
			logger.LogError("tunnel_target_to_client", err)
		}
		out.Store(n)
		logger.Trace("tunnel_transfer_complete", "direction", "target_to_client", "bytes", n)
		// Signal EOF to client
		if tc, ok := client.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	wg.Wait()
	logger.Trace("tunnel_closed", "client", client.RemoteAddr(), "target", target.RemoteAddr(), "bytes_in", in.Load(), "bytes_out", out.Load())
	return in.Load(), out.Load()
}

// copyWithIdleTimeout copies from src to dst, resetting the deadline after each successful read.
func copyWithIdleTimeout(dst, src net.Conn, idleTimeout time.Duration) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var total int64

	for {
		// Set read deadline
		src.SetReadDeadline(time.Now().Add(idleTimeout))

		n, readErr := src.Read(buf)
		if n > 0 {
			// Reset write deadline on successful read
			dst.SetWriteDeadline(time.Now().Add(idleTimeout))

			written, writeErr := dst.Write(buf[:n])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return total, nil
			}
			return total, readErr
		}
	}
}

// isTimeoutError checks if the error is a timeout error.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}
