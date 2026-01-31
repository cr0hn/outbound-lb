// Package health provides IP health checking functionality.
package health

import (
	"context"
	"sync"
	"time"

	"github.com/cr0hn/outbound-lb/internal/logger"
	"github.com/cr0hn/outbound-lb/internal/metrics"
)

// Checker is the interface for health check implementations.
type Checker interface {
	// Check performs a health check from the given source IP.
	// Returns nil if the check succeeds, error otherwise.
	Check(ctx context.Context, sourceIP string) error
}

// HealthCheckerConfig holds configuration for the HealthChecker.
type HealthCheckerConfig struct {
	IPs              []string
	Checker          Checker
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold int
	SuccessThreshold int
}

// HealthChecker manages health checking for multiple IPs.
type HealthChecker struct {
	config   HealthCheckerConfig
	statuses map[string]*IPStatus
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(cfg HealthCheckerConfig) *HealthChecker {
	hc := &HealthChecker{
		config:   cfg,
		statuses: make(map[string]*IPStatus, len(cfg.IPs)),
		stopCh:   make(chan struct{}),
	}

	for _, ip := range cfg.IPs {
		hc.statuses[ip] = NewIPStatus(ip)
		// Initialize metrics
		metrics.IPHealthStatus.WithLabelValues(ip).Set(1) // Start as healthy
	}

	return hc
}

// Start starts the health check goroutine.
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.checkLoop()
	logger.Info("health_checker_started",
		"interval", hc.config.Interval,
		"timeout", hc.config.Timeout,
		"failure_threshold", hc.config.FailureThreshold,
		"success_threshold", hc.config.SuccessThreshold,
	)
}

// Stop stops the health checker and waits for completion.
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
	logger.Info("health_checker_stopped")
}

// IsHealthy returns true if the IP is in a healthy state.
func (hc *HealthChecker) IsHealthy(ip string) bool {
	hc.mu.RLock()
	status, ok := hc.statuses[ip]
	hc.mu.RUnlock()

	if !ok {
		return true // Unknown IPs are considered healthy
	}
	return status.IsHealthy()
}

// healthyIPsPool is a pool for slices used in GetHealthyIPs to reduce allocations.
var healthyIPsPool = sync.Pool{
	New: func() any {
		s := make([]string, 0, 64)
		return &s
	},
}

// GetHealthyIPs filters the given IPs and returns only the healthy ones.
// Returns a slice from a pool; the caller should not retain the slice.
func (hc *HealthChecker) GetHealthyIPs(ips []string) []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Get slice from pool
	resultPtr := healthyIPsPool.Get().(*[]string)
	result := (*resultPtr)[:0]

	for _, ip := range ips {
		status, ok := hc.statuses[ip]
		if !ok || status.IsHealthy() {
			result = append(result, ip)
		}
	}

	// Return the slice to the pool eventually via a finalizer or manual release
	// For simplicity, we just return the result - it will be GC'd
	return result
}

// GetAllStatus returns status info for all IPs.
func (hc *HealthChecker) GetAllStatus() []StatusInfo {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make([]StatusInfo, 0, len(hc.statuses))
	for _, status := range hc.statuses {
		result = append(result, status.GetInfo())
	}
	return result
}

// checkLoop runs periodic health checks.
func (hc *HealthChecker) checkLoop() {
	defer hc.wg.Done()

	// Run an initial check immediately
	hc.checkAll()

	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkAll()
		case <-hc.stopCh:
			return
		}
	}
}

// checkAll performs health checks on all IPs.
func (hc *HealthChecker) checkAll() {
	var wg sync.WaitGroup

	hc.mu.RLock()
	ips := make([]string, 0, len(hc.statuses))
	for ip := range hc.statuses {
		ips = append(ips, ip)
	}
	hc.mu.RUnlock()

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			hc.checkIP(ip)
		}(ip)
	}

	wg.Wait()
	hc.updateAggregateMetrics()
}

// checkIP performs a health check on a single IP.
func (hc *HealthChecker) checkIP(ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	start := time.Now()
	err := hc.config.Checker.Check(ctx, ip)
	duration := time.Since(start)

	// Record metrics
	metrics.HealthCheckDuration.WithLabelValues(ip).Observe(duration.Seconds())

	hc.mu.RLock()
	status, ok := hc.statuses[ip]
	hc.mu.RUnlock()

	if !ok {
		return
	}

	if err != nil {
		metrics.HealthCheckTotal.WithLabelValues(ip, "failure").Inc()
		changed := status.RecordFailure(err, hc.config.FailureThreshold)
		if changed {
			newState := status.GetState()
			logger.Warn("ip_health_state_changed",
				"ip", ip,
				"state", newState.String(),
				"error", err.Error(),
			)
			if newState == StateUnhealthy {
				metrics.IPHealthStatus.WithLabelValues(ip).Set(0)
			}
		} else {
			logger.Debug("health_check_failed",
				"ip", ip,
				"error", err.Error(),
				"consecutive_failures", status.ConsecutiveFailures,
			)
		}
	} else {
		metrics.HealthCheckTotal.WithLabelValues(ip, "success").Inc()
		changed := status.RecordSuccess(hc.config.SuccessThreshold)
		if changed {
			newState := status.GetState()
			logger.Info("ip_health_state_changed",
				"ip", ip,
				"state", newState.String(),
			)
			if newState == StateHealthy {
				metrics.IPHealthStatus.WithLabelValues(ip).Set(1)
			}
		} else {
			logger.Trace("health_check_success", "ip", ip, "duration", duration)
		}
	}
}

// updateAggregateMetrics updates the aggregate health metrics.
func (hc *HealthChecker) updateAggregateMetrics() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy, unhealthy int
	for _, status := range hc.statuses {
		if status.IsHealthy() {
			healthy++
		} else {
			unhealthy++
		}
	}

	metrics.HealthyIPs.Set(float64(healthy))
	metrics.UnhealthyIPs.Set(float64(unhealthy))
}
