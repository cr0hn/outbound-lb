// Package proxy provides the HTTP/HTTPS proxy server.
package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cr0hn/outbound-lb/internal/logger"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// hopByHopHeaders contains headers that should not be forwarded to the upstream server.
// Defined as package-level variable to avoid allocation on each request.
var hopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Proxy-Connection":    true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

// hopByHopHeadersList is the list form for deletion operations.
var hopByHopHeadersList = []string{
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

// Handler handles HTTP proxy requests.
type Handler struct {
	server *Server
}

// NewHandler creates a new Handler.
func NewHandler(server *Server) *Handler {
	return &Handler{server: server}
}

// ServeHTTP handles an HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Generate request ID for tracing
	requestID := GenerateRequestID()

	// Create cancellable context with request ID
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	ctx = ContextWithRequestID(ctx, requestID)

	// Update request with new context
	r = r.WithContext(ctx)

	logger.Trace("request_received", "request_id", requestID, "method", r.Method, "host", r.Host, "remote", r.RemoteAddr, "url", r.URL.String())

	// Check authentication
	if !h.server.authenticate(w, r) {
		logger.Trace("request_auth_failed", "remote", r.RemoteAddr)
		return
	}

	// CONNECT requests are handled separately
	if r.Method == http.MethodConnect {
		h.server.connectHandler.ServeHTTP(w, r)
		return
	}

	// Get the host
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	logger.Trace("ip_selection_start", "host", host)

	// Select outbound IP
	ip, err := h.server.selectIP(host)
	if err != nil {
		logger.Trace("ip_selection_failed", "host", host, "error", err)
		h.sendError(w, http.StatusServiceUnavailable, "No available outbound IPs")
		metrics.LimitRejections.WithLabelValues("total").Inc()
		return
	}

	logger.Trace("ip_selected", "host", host, "ip", ip)

	// Acquire connection slot
	logger.Trace("connection_acquire_attempt", "ip", ip)
	if err := h.server.limiter.Acquire(ip); err != nil {
		logger.Trace("connection_acquire_failed", "ip", ip, "error", err)
		h.sendError(w, http.StatusServiceUnavailable, "Connection limit reached")
		metrics.LimitRejections.WithLabelValues("per_ip").Inc()
		logger.LogConnectionLimit("per_ip", ip, int(h.server.limiter.GetIPCount(ip)), h.server.cfg.MaxConnsPerIP)
		return
	}
	logger.Trace("connection_acquired", "ip", ip)
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

	// Get transport for this IP
	transport := h.server.transportPool.Get(ip)

	// Create outgoing request
	outReq := h.createOutgoingRequest(r)

	// Execute request
	logger.Trace("upstream_request_start", "host", host, "ip", ip, "method", r.Method)
	resp, err := transport.RoundTrip(outReq)
	if err != nil {
		logger.Trace("upstream_request_failed", "host", host, "ip", ip, "error", err)
		logger.LogError("proxy_request", err, "host", host, "ip", ip)
		h.sendError(w, http.StatusBadGateway, "Failed to connect to upstream")
		metrics.RequestsTotal.WithLabelValues(r.Method, "502").Inc()
		return
	}
	defer resp.Body.Close()

	logger.Trace("upstream_response_received", "host", host, "ip", ip, "status", resp.StatusCode)

	// Copy response headers
	h.copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	bytesCopied, err := io.Copy(w, resp.Body)
	if err != nil {
		// Cannot send error to client - headers already sent
		logger.LogError("response_copy", err, "host", host, "ip", ip)
	}

	logger.Trace("response_copy_complete", "host", host, "ip", ip, "bytes", bytesCopied)

	// Log and record metrics
	duration := time.Since(start).Milliseconds()
	logger.LogRequest(r.Method, host, r.RemoteAddr, ip, resp.StatusCode, duration, r.ContentLength, bytesCopied)

	h.server.stats.IncTotalRequests()
	h.server.stats.AddBytesSent(bytesCopied)
	if r.ContentLength > 0 {
		h.server.stats.AddBytesReceived(r.ContentLength)
	}

	metrics.RequestsTotal.WithLabelValues(r.Method, fmt.Sprintf("%d", resp.StatusCode)).Inc()
	metrics.RequestDuration.WithLabelValues(r.Method).Observe(time.Since(start).Seconds())
}

// createOutgoingRequest creates the outgoing request from the incoming request.
func (h *Handler) createOutgoingRequest(r *http.Request) *http.Request {
	outReq := r.Clone(r.Context())

	// For proxy requests, the URL must be absolute
	if !outReq.URL.IsAbs() {
		outReq.URL.Scheme = "http"
		outReq.URL.Host = r.Host
	}

	// Remove hop-by-hop headers
	h.removeHopByHopHeaders(outReq.Header)

	// Set X-Forwarded-For
	if clientIP := h.getClientIP(r); clientIP != "" {
		if prior := outReq.Header.Get("X-Forwarded-For"); prior != "" {
			outReq.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			outReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	return outReq
}

// copyHeaders copies headers from src to dst.
func (h *Handler) copyHeaders(dst, src http.Header) {
	for key, values := range src {
		// Skip hop-by-hop headers
		if isHopByHop(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// removeHopByHopHeaders removes hop-by-hop headers from the request.
func (h *Handler) removeHopByHopHeaders(header http.Header) {
	for _, hdr := range hopByHopHeadersList {
		header.Del(hdr)
	}

	// Also remove headers listed in Connection header
	if conn := header.Get("Connection"); conn != "" {
		for _, h := range strings.Split(conn, ",") {
			header.Del(strings.TrimSpace(h))
		}
	}
}

// isHopByHop returns true if the header is a hop-by-hop header.
func isHopByHop(header string) bool {
	return hopByHopHeaders[header]
}

// getClientIP extracts the client IP from the request.
func (h *Handler) getClientIP(r *http.Request) string {
	// Handle IPv6 addresses in brackets [::1]:port
	if strings.HasPrefix(r.RemoteAddr, "[") {
		if idx := strings.LastIndex(r.RemoteAddr, "]:"); idx != -1 {
			return r.RemoteAddr[1:idx]
		}
		return r.RemoteAddr
	}
	// Handle IPv4 addresses host:port
	host, _, found := strings.Cut(r.RemoteAddr, ":")
	if found {
		return host
	}
	return r.RemoteAddr
}

// sendError sends an error response.
func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}
