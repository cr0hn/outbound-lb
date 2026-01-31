package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// HTTPChecker implements health checking via HTTP request.
type HTTPChecker struct {
	url     string // Full URL (e.g., "http://httpbin.org/status/200")
	timeout time.Duration
}

// NewHTTPChecker creates a new HTTP health checker.
func NewHTTPChecker(url string, timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		url:     url,
		timeout: timeout,
	}
}

// Check performs an HTTP GET health check from the given source IP.
func (c *HTTPChecker) Check(ctx context.Context, sourceIP string) error {
	// Create a transport with the source IP bound
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP: net.ParseIP(sourceIP),
				},
				Timeout: c.timeout,
			}
			return dialer.DialContext(ctx, network, addr)
		},
		DisableKeepAlives: true, // Don't keep connections for health checks
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   c.timeout,
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx status codes as success
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
