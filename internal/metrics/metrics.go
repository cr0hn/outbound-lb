// Package metrics provides Prometheus metrics for the proxy.
package metrics

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts total proxy requests by status.
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "outbound_lb_requests_total",
		Help: "Total number of proxy requests",
	}, []string{"method", "status"})

	// RequestDuration tracks request duration in seconds.
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "outbound_lb_request_duration_seconds",
		Help:    "Request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})

	// BytesSent tracks total bytes sent to clients.
	BytesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbound_lb_bytes_sent_total",
		Help: "Total bytes sent to clients",
	})

	// BytesReceived tracks total bytes received from clients.
	BytesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbound_lb_bytes_received_total",
		Help: "Total bytes received from clients",
	})

	// ActiveConnections tracks current active connections.
	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbound_lb_active_connections",
		Help: "Current number of active connections",
	})

	// ConnectionsPerIP tracks connections per outbound IP.
	ConnectionsPerIP = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "outbound_lb_connections_per_ip",
		Help: "Current connections per outbound IP",
	}, []string{"ip"})

	// BalancerSelections tracks IP selections by the balancer.
	BalancerSelections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "outbound_lb_balancer_selections_total",
		Help: "Total IP selections by the balancer",
	}, []string{"ip", "host"})

	// LimitRejections tracks connection rejections due to limits.
	LimitRejections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "outbound_lb_limit_rejections_total",
		Help: "Total connection rejections due to limits",
	}, []string{"type"})

	// AuthFailures tracks authentication failures.
	AuthFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbound_lb_auth_failures_total",
		Help: "Total authentication failures",
	})

	// TunnelConnections tracks CONNECT tunnel connections.
	TunnelConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbound_lb_tunnel_connections_total",
		Help: "Total CONNECT tunnel connections",
	})

	// HistoryEntries tracks entries in the balancer history.
	HistoryEntries = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbound_lb_history_entries",
		Help: "Current number of entries in balancer history",
	})

	// HistoryHosts tracks unique hosts in the balancer history.
	HistoryHosts = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbound_lb_history_hosts",
		Help: "Current number of unique hosts in balancer history",
	})

	// Health check metrics

	// HealthCheckTotal counts total health checks by IP and result.
	HealthCheckTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "outbound_lb_health_check_total",
		Help: "Total health checks by IP and result",
	}, []string{"ip", "result"}) // result: "success" or "failure"

	// IPHealthStatus tracks current health status per IP (1=healthy, 0=unhealthy).
	IPHealthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "outbound_lb_ip_health_status",
		Help: "Health status per IP (1=healthy, 0=unhealthy)",
	}, []string{"ip"})

	// HealthCheckDuration tracks health check duration.
	HealthCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "outbound_lb_health_check_duration_seconds",
		Help:    "Health check duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	}, []string{"ip"})

	// HealthyIPs tracks the number of healthy IPs.
	HealthyIPs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbound_lb_healthy_ips",
		Help: "Number of healthy IPs",
	})

	// UnhealthyIPs tracks the number of unhealthy IPs.
	UnhealthyIPs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbound_lb_unhealthy_ips",
		Help: "Number of unhealthy IPs",
	})
)

// Stats holds runtime statistics for the /stats endpoint.
type Stats struct {
	ActiveConnections int64             `json:"active_connections"`
	TotalRequests     int64             `json:"total_requests"`
	BytesSent         int64             `json:"bytes_sent"`
	BytesReceived     int64             `json:"bytes_received"`
	ConnectionsPerIP  map[string]int64  `json:"connections_per_ip"`
	SelectionsPerIP   map[string]int64  `json:"selections_per_ip"`
}

// StatsCollector collects runtime statistics.
type StatsCollector struct {
	activeConnections atomic.Int64
	totalRequests     atomic.Int64
	bytesSent         atomic.Int64
	bytesReceived     atomic.Int64
	connectionsPerIP  map[string]*atomic.Int64
	selectionsPerIP   map[string]*atomic.Int64
}

// NewStatsCollector creates a new stats collector.
func NewStatsCollector(ips []string) *StatsCollector {
	sc := &StatsCollector{
		connectionsPerIP: make(map[string]*atomic.Int64),
		selectionsPerIP:  make(map[string]*atomic.Int64),
	}
	for _, ip := range ips {
		sc.connectionsPerIP[ip] = &atomic.Int64{}
		sc.selectionsPerIP[ip] = &atomic.Int64{}
	}
	return sc
}

// IncActiveConnections increments active connections.
func (sc *StatsCollector) IncActiveConnections() {
	sc.activeConnections.Add(1)
	ActiveConnections.Inc()
}

// DecActiveConnections decrements active connections.
func (sc *StatsCollector) DecActiveConnections() {
	sc.activeConnections.Add(-1)
	ActiveConnections.Dec()
}

// IncTotalRequests increments total requests.
func (sc *StatsCollector) IncTotalRequests() {
	sc.totalRequests.Add(1)
}

// AddBytesSent adds to bytes sent counter.
func (sc *StatsCollector) AddBytesSent(n int64) {
	sc.bytesSent.Add(n)
	BytesSent.Add(float64(n))
}

// AddBytesReceived adds to bytes received counter.
func (sc *StatsCollector) AddBytesReceived(n int64) {
	sc.bytesReceived.Add(n)
	BytesReceived.Add(float64(n))
}

// IncConnectionsForIP increments connections for an IP.
func (sc *StatsCollector) IncConnectionsForIP(ip string) {
	if counter, ok := sc.connectionsPerIP[ip]; ok {
		counter.Add(1)
	}
	ConnectionsPerIP.WithLabelValues(ip).Inc()
}

// DecConnectionsForIP decrements connections for an IP.
func (sc *StatsCollector) DecConnectionsForIP(ip string) {
	if counter, ok := sc.connectionsPerIP[ip]; ok {
		counter.Add(-1)
	}
	ConnectionsPerIP.WithLabelValues(ip).Dec()
}

// IncSelectionsForIP increments selections for an IP.
func (sc *StatsCollector) IncSelectionsForIP(ip, host string) {
	if counter, ok := sc.selectionsPerIP[ip]; ok {
		counter.Add(1)
	}
	BalancerSelections.WithLabelValues(ip, host).Inc()
}

// GetStats returns current statistics.
func (sc *StatsCollector) GetStats() Stats {
	connsPerIP := make(map[string]int64)
	for ip, counter := range sc.connectionsPerIP {
		connsPerIP[ip] = counter.Load()
	}
	selsPerIP := make(map[string]int64)
	for ip, counter := range sc.selectionsPerIP {
		selsPerIP[ip] = counter.Load()
	}
	return Stats{
		ActiveConnections: sc.activeConnections.Load(),
		TotalRequests:     sc.totalRequests.Load(),
		BytesSent:         sc.bytesSent.Load(),
		BytesReceived:     sc.bytesReceived.Load(),
		ConnectionsPerIP:  connsPerIP,
		SelectionsPerIP:   selsPerIP,
	}
}
