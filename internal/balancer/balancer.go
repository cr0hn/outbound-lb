// Package balancer provides IP load balancing algorithms.
package balancer

import "time"

// Balancer is the interface for IP selection algorithms.
type Balancer interface {
	// Select returns the best IP to use for the given host.
	Select(host string) (string, error)
	// Record records that an IP was used for a host.
	Record(host, ip string)
	// GetStats returns balancer statistics.
	GetStats() Stats
	// Start starts background goroutines.
	Start()
	// Stop stops background goroutines.
	Stop()
	// UpdateHistoryConfig updates history configuration at runtime.
	UpdateHistoryConfig(window time.Duration, size int)
}

// Stats holds balancer statistics.
type Stats struct {
	TotalHosts    int            `json:"total_hosts"`
	TotalEntries  int            `json:"total_entries"`
	EntriesPerIP  map[string]int `json:"entries_per_ip"`
}

// Config holds balancer configuration.
type Config struct {
	IPs           []string
	HistoryWindow int64 // in seconds
	HistorySize   int
	Limiter       IPLimiter
	HealthChecker IPHealthChecker
}

// IPLimiter is the interface for checking IP availability.
type IPLimiter interface {
	IsIPAvailable(ip string) bool
	// GetAvailableIPs returns IPs that have available connection slots.
	// The returned slice is borrowed from a pool; caller MUST call ReleaseAvailableIPs
	// when done with the slice to return it to the pool.
	GetAvailableIPs(ips []string) []string
}

// IPHealthChecker is the interface for checking IP health status.
type IPHealthChecker interface {
	// IsHealthy returns true if the IP is healthy.
	IsHealthy(ip string) bool
	// GetHealthyIPs filters the given IPs and returns only the healthy ones.
	GetHealthyIPs(ips []string) []string
}

// New creates a new LRU balancer.
func New(cfg Config) Balancer {
	return NewLRU(cfg)
}
