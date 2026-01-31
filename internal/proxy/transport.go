// Package proxy provides the HTTP/HTTPS proxy server.
package proxy

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// TransportPool manages http.Transport instances per outbound IP.
type TransportPool struct {
	transports map[string]*http.Transport
	timeout    time.Duration
	mu         sync.RWMutex
}

// NewTransportPool creates a new transport pool.
func NewTransportPool(ips []string, timeout time.Duration) *TransportPool {
	tp := &TransportPool{
		transports: make(map[string]*http.Transport),
		timeout:    timeout,
	}

	for _, ip := range ips {
		tp.transports[ip] = tp.createTransport(ip)
	}

	return tp
}

// Get returns the transport for the given IP.
func (tp *TransportPool) Get(ip string) *http.Transport {
	tp.mu.RLock()
	t, exists := tp.transports[ip]
	tp.mu.RUnlock()

	if exists {
		return t
	}

	// Create transport for unknown IP
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if t, exists := tp.transports[ip]; exists {
		return t
	}

	t = tp.createTransport(ip)
	tp.transports[ip] = t
	return t
}

// createTransport creates a new http.Transport bound to the given IP.
func (tp *TransportPool) createTransport(ip string) *http.Transport {
	localAddr := &net.TCPAddr{
		IP: net.ParseIP(ip),
	}

	dialer := &net.Dialer{
		LocalAddr: localAddr,
		Timeout:   tp.timeout,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

// Close closes all transports.
func (tp *TransportPool) Close() {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	for _, t := range tp.transports {
		t.CloseIdleConnections()
	}
}

// Dialer creates connections bound to a specific outbound IP.
type Dialer struct {
	localIP     string
	timeout     time.Duration
	idleTimeout time.Duration
}

// NewDialer creates a new Dialer.
func NewDialer(localIP string, timeout, idleTimeout time.Duration) *Dialer {
	return &Dialer{
		localIP:     localIP,
		timeout:     timeout,
		idleTimeout: idleTimeout,
	}
}

// Dial creates a connection to the given address.
func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

// DialContext creates a connection to the given address with context.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	localAddr := &net.TCPAddr{
		IP: net.ParseIP(d.localIP),
	}

	dialer := &net.Dialer{
		LocalAddr: localAddr,
		Timeout:   d.timeout,
		KeepAlive: 30 * time.Second,
	}

	return dialer.DialContext(ctx, network, addr)
}
