// Package balancer provides IP load balancing algorithms.
package balancer

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/cr0hn/outbound-lb/internal/logger"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

var (
	// ErrNoAvailableIPs is returned when no IPs are available.
	ErrNoAvailableIPs = errors.New("no available IPs")
)

// selectContext holds temporary maps used during Select() to avoid allocations.
type selectContext struct {
	usageCount map[string]int
	lastUsed   map[string]time.Time
}

// selectContextPool is a sync.Pool for reusing selectContext in Select().
// This significantly reduces allocations in the hot path.
var selectContextPool = sync.Pool{
	New: func() any {
		return &selectContext{
			usageCount: make(map[string]int, 16),
			lastUsed:   make(map[string]time.Time, 16),
		}
	},
}

// LRU implements the Least Recently Used per Host algorithm.
type LRU struct {
	ips           []string
	historyWindow time.Duration
	historySize   int
	limiter       IPLimiter
	healthChecker IPHealthChecker
	history       *History
	stopCh        chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

// NewLRU creates a new LRU balancer.
func NewLRU(cfg Config) *LRU {
	return &LRU{
		ips:           cfg.IPs,
		historyWindow: time.Duration(cfg.HistoryWindow) * time.Second,
		historySize:   cfg.HistorySize,
		limiter:       cfg.Limiter,
		healthChecker: cfg.HealthChecker,
		history:       NewHistory(),
		stopCh:        make(chan struct{}),
	}
}

// UpdateHistoryConfig updates the history configuration at runtime.
func (l *LRU) UpdateHistoryConfig(window time.Duration, size int) {
	l.mu.Lock()
	l.historyWindow = window
	l.historySize = size
	l.mu.Unlock()
	logger.Info("history_config_updated", "window", window, "size", size)
}

// Start starts the background cleanup goroutine.
func (l *LRU) Start() {
	l.wg.Add(1)
	go l.cleanupLoop()
}

// Stop stops the background cleanup goroutine.
func (l *LRU) Stop() {
	close(l.stopCh)
	l.wg.Wait()
}

// cleanupLoop periodically cleans up expired history entries.
func (l *LRU) cleanupLoop() {
	defer l.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.RLock()
			window := l.historyWindow
			l.mu.RUnlock()

			removedEntries, removedHosts := l.history.Cleanup(window)
			if removedEntries > 0 || removedHosts > 0 {
				// Update metrics
				hosts, entries, _ := l.history.Stats()
				metrics.HistoryHosts.Set(float64(hosts))
				metrics.HistoryEntries.Set(float64(entries))
			}
		case <-l.stopCh:
			return
		}
	}
}

// Select returns the best IP to use for the given host.
// Algorithm:
// 1. Get history for the host within window and size limits
// 2. Count usage per IP in the filtered history
// 3. Exclude IPs that have reached connection limits
// 4. Select IP with lowest usage count (tie-break by oldest last use)
func (l *LRU) Select(host string) (string, error) {
	logger.Trace("balancer_select_start", "host", host)

	// Get available IPs (not at connection limit)
	availableIPs := l.getAvailableIPs()
	if len(availableIPs) == 0 {
		logger.Trace("balancer_no_available_ips", "host", host, "total_ips", len(l.ips))
		return "", ErrNoAvailableIPs
	}

	logger.Trace("balancer_available_ips", "host", host, "count", len(availableIPs), "ips", availableIPs)

	// Get history config under lock
	l.mu.RLock()
	window := l.historyWindow
	size := l.historySize
	l.mu.RUnlock()

	// Get filtered history for this host
	entries := l.history.GetFiltered(host, window, size)
	logger.Trace("balancer_history_entries", "host", host, "count", len(entries), "window", window, "max_size", size)

	// Get context from pool to avoid allocations
	ctx := selectContextPool.Get().(*selectContext)
	defer func() {
		// Clear maps and return to pool
		clear(ctx.usageCount)
		clear(ctx.lastUsed)
		selectContextPool.Put(ctx)
	}()

	// Count usage per IP and track last use time
	for _, e := range entries {
		ctx.usageCount[e.IP]++
		if t, exists := ctx.lastUsed[e.IP]; !exists || e.Timestamp.After(t) {
			ctx.lastUsed[e.IP] = e.Timestamp
		}
	}

	// Find IP with lowest usage among available IPs
	var selectedIP string
	minUsage := math.MaxInt
	var oldestUse time.Time

	for _, ip := range availableIPs {
		usage := ctx.usageCount[ip]
		lastUse := ctx.lastUsed[ip]

		if usage < minUsage {
			minUsage = usage
			selectedIP = ip
			oldestUse = lastUse
		} else if usage == minUsage {
			// Tie-break: prefer IP with oldest last use (or never used)
			if lastUse.IsZero() || lastUse.Before(oldestUse) {
				selectedIP = ip
				oldestUse = lastUse
			}
		}
	}

	logger.Trace("balancer_selection_complete", "host", host, "selected", selectedIP, "usage_count", minUsage, "usage_counts", ctx.usageCount)
	return selectedIP, nil
}

// Record records that an IP was used for a host.
func (l *LRU) Record(host, ip string) {
	l.history.Record(host, ip)

	// Update metrics
	hosts, entries, _ := l.history.Stats()
	metrics.HistoryHosts.Set(float64(hosts))
	metrics.HistoryEntries.Set(float64(entries))
}

// GetStats returns balancer statistics.
func (l *LRU) GetStats() Stats {
	hosts, entries, entriesPerIP := l.history.Stats()
	return Stats{
		TotalHosts:   hosts,
		TotalEntries: entries,
		EntriesPerIP: entriesPerIP,
	}
}

// getAvailableIPs returns IPs that are healthy and haven't reached connection limits.
// Applies health check filter first, then limiter filter.
// Implements graceful degradation: if all IPs are unhealthy, uses all IPs.
func (l *LRU) getAvailableIPs() []string {
	ips := l.ips

	// 1. Filter by health check (if configured)
	if l.healthChecker != nil {
		healthyIPs := l.healthChecker.GetHealthyIPs(ips)
		// Graceful degradation: if all IPs are unhealthy, use all
		if len(healthyIPs) == 0 {
			logger.Warn("all_ips_unhealthy", "using_all", true, "total_ips", len(ips))
		} else {
			ips = healthyIPs
		}
	}

	// 2. Filter by limiter (connection limits)
	if l.limiter != nil {
		return l.limiter.GetAvailableIPs(ips)
	}
	return ips
}

// releaseAvailableIPs releases a slice obtained from getAvailableIPs back to the pool.
// Only call this if limiter is configured.
func (l *LRU) releaseAvailableIPs(s []string) {
	if l.limiter != nil && cap(s) <= 64 {
		// Return to pool - we can't import limiter package, so we just let it be GC'd
		// The limiter.ReleaseAvailableIPs function handles pooling
	}
}
