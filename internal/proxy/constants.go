// Package proxy provides the HTTP/HTTPS proxy server.
package proxy

import "time"

// Default timeouts and intervals.
const (
	// DefaultCleanupInterval is the interval for cleaning up expired history entries.
	DefaultCleanupInterval = 30 * time.Second

	// DefaultDebounceInterval is the interval for debouncing config updates.
	DefaultDebounceInterval = 100 * time.Millisecond

	// DefaultShutdownTimeout is the timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second

	// DefaultTCPKeepAlive is the TCP keep-alive interval for connections.
	DefaultTCPKeepAlive = 30 * time.Second

	// DefaultIdleConnTimeout is the timeout for idle HTTP connections.
	DefaultIdleConnTimeout = 90 * time.Second

	// DefaultTLSHandshakeTimeout is the timeout for TLS handshakes.
	DefaultTLSHandshakeTimeout = 10 * time.Second

	// DefaultExpectContinueTimeout is the timeout for 100-continue responses.
	DefaultExpectContinueTimeout = 1 * time.Second
)

// Pool and buffer sizes.
const (
	// DefaultPoolCapacity is the default initial capacity for pooled slices.
	DefaultPoolCapacity = 16

	// DefaultMaxPooledSliceSize is the maximum size of slices to return to pools.
	DefaultMaxPooledSliceSize = 64

	// DefaultTunnelBufferSize is the buffer size for tunnel copy operations.
	DefaultTunnelBufferSize = 32 * 1024 // 32KB
)

// Transport limits.
const (
	// DefaultMaxIdleConns is the maximum number of idle connections across all hosts.
	DefaultMaxIdleConns = 100

	// DefaultMaxIdleConnsPerHost is the maximum number of idle connections per host.
	DefaultMaxIdleConnsPerHost = 10
)
