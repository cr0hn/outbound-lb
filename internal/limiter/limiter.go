// Package limiter provides connection limiting functionality.
package limiter

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/cr0hn/outbound-lb/internal/logger"
)

var (
	// ErrIPLimitReached is returned when the per-IP limit is reached.
	ErrIPLimitReached = errors.New("connection limit reached for IP")
	// ErrTotalLimitReached is returned when the total limit is reached.
	ErrTotalLimitReached = errors.New("total connection limit reached")
)

// availableIPsPool is a sync.Pool for reusing slices in GetAvailableIPs.
// This reduces allocations in the hot path.
var availableIPsPool = sync.Pool{
	New: func() any {
		return make([]string, 0, 16)
	},
}

// Limiter tracks and limits concurrent connections.
type Limiter struct {
	maxPerIP atomic.Int32
	maxTotal atomic.Int32
	total    atomic.Int64
	perIP    map[string]*atomic.Int64
	mu       sync.RWMutex
}

// New creates a new Limiter.
func New(maxPerIP, maxTotal int, ips []string) *Limiter {
	l := &Limiter{
		perIP: make(map[string]*atomic.Int64),
	}
	l.maxPerIP.Store(int32(maxPerIP))
	l.maxTotal.Store(int32(maxTotal))
	for _, ip := range ips {
		l.perIP[ip] = &atomic.Int64{}
	}
	return l
}

// UpdateLimits updates the connection limits at runtime.
func (l *Limiter) UpdateLimits(maxPerIP, maxTotal int) {
	l.maxPerIP.Store(int32(maxPerIP))
	l.maxTotal.Store(int32(maxTotal))
	logger.Info("limits_updated", "max_per_ip", maxPerIP, "max_total", maxTotal)
}

// Acquire attempts to acquire a connection slot for the given IP.
// Returns nil if successful, error if limit reached.
// Uses CAS loops to prevent TOCTOU race conditions.
func (l *Limiter) Acquire(ip string) error {
	maxTotal := int64(l.maxTotal.Load())
	maxPerIP := int64(l.maxPerIP.Load())

	// Atomically increment total counter with CAS loop
	for {
		current := l.total.Load()
		if current >= maxTotal {
			return ErrTotalLimitReached
		}
		if l.total.CompareAndSwap(current, current+1) {
			break
		}
	}

	// Get or create per-IP counter
	l.mu.RLock()
	counter, exists := l.perIP[ip]
	l.mu.RUnlock()

	if !exists {
		l.mu.Lock()
		if _, exists := l.perIP[ip]; !exists {
			l.perIP[ip] = &atomic.Int64{}
		}
		counter = l.perIP[ip]
		l.mu.Unlock()
	}

	// Atomically increment per-IP counter with CAS loop
	for {
		ipCount := counter.Load()
		if ipCount >= maxPerIP {
			// Rollback total counter since we can't acquire
			l.total.Add(-1)
			return ErrIPLimitReached
		}
		if counter.CompareAndSwap(ipCount, ipCount+1) {
			break
		}
	}

	return nil
}

// Release releases a connection slot for the given IP.
func (l *Limiter) Release(ip string) {
	l.mu.RLock()
	counter, exists := l.perIP[ip]
	l.mu.RUnlock()

	if exists {
		counter.Add(-1)
	}
	l.total.Add(-1)
}

// GetIPCount returns the current connection count for an IP.
func (l *Limiter) GetIPCount(ip string) int64 {
	l.mu.RLock()
	counter, exists := l.perIP[ip]
	l.mu.RUnlock()

	if !exists {
		return 0
	}
	return counter.Load()
}

// GetTotalCount returns the current total connection count.
func (l *Limiter) GetTotalCount() int64 {
	return l.total.Load()
}

// IsIPAvailable checks if an IP has available connection slots.
func (l *Limiter) IsIPAvailable(ip string) bool {
	l.mu.RLock()
	counter, exists := l.perIP[ip]
	l.mu.RUnlock()

	if !exists {
		return true
	}
	return counter.Load() < int64(l.maxPerIP.Load())
}

// GetAvailableIPs returns IPs that have available connection slots.
// The returned slice is borrowed from a pool; caller MUST call ReleaseAvailableIPs
// when done with the slice to return it to the pool.
func (l *Limiter) GetAvailableIPs(ips []string) []string {
	available := availableIPsPool.Get().([]string)
	available = available[:0] // Reset length without reallocating

	for _, ip := range ips {
		if l.IsIPAvailable(ip) {
			available = append(available, ip)
		}
	}
	return available
}

// ReleaseAvailableIPs returns a slice obtained from GetAvailableIPs back to the pool.
// The slice should not be used after calling this function.
func ReleaseAvailableIPs(s []string) {
	if cap(s) <= 64 { // Don't pool very large slices
		availableIPsPool.Put(s[:0])
	}
}

// Stats returns current limiter statistics.
func (l *Limiter) Stats() map[string]int64 {
	stats := make(map[string]int64)
	stats["total"] = l.total.Load()

	l.mu.RLock()
	for ip, counter := range l.perIP {
		stats[ip] = counter.Load()
	}
	l.mu.RUnlock()

	return stats
}
