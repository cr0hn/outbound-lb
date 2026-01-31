package health

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TCPChecker implements health checking via TCP connection.
type TCPChecker struct {
	target  string // host:port (e.g., "1.1.1.1:443")
	timeout time.Duration
}

// NewTCPChecker creates a new TCP health checker.
func NewTCPChecker(target string, timeout time.Duration) *TCPChecker {
	return &TCPChecker{
		target:  target,
		timeout: timeout,
	}
}

// Check performs a TCP connection health check from the given source IP.
func (c *TCPChecker) Check(ctx context.Context, sourceIP string) error {
	// Create a dialer with the source IP
	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{
			IP: net.ParseIP(sourceIP),
		},
		Timeout: c.timeout,
	}

	// Dial with context
	conn, err := dialer.DialContext(ctx, "tcp", c.target)
	if err != nil {
		return fmt.Errorf("tcp connect failed: %w", err)
	}
	defer conn.Close()

	return nil
}
