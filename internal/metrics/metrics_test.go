package metrics

import (
	"testing"
)

func TestNewStatsCollector(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.1.2"}
	sc := NewStatsCollector(ips)

	if sc == nil {
		t.Fatal("expected non-nil stats collector")
	}

	if len(sc.connectionsPerIP) != 2 {
		t.Errorf("expected 2 IPs in connectionsPerIP, got %d", len(sc.connectionsPerIP))
	}
}

func TestStatsCollector_ActiveConnections(t *testing.T) {
	sc := NewStatsCollector([]string{"192.168.1.1"})

	sc.IncActiveConnections()
	sc.IncActiveConnections()

	stats := sc.GetStats()
	if stats.ActiveConnections != 2 {
		t.Errorf("expected 2 active connections, got %d", stats.ActiveConnections)
	}

	sc.DecActiveConnections()
	stats = sc.GetStats()
	if stats.ActiveConnections != 1 {
		t.Errorf("expected 1 active connection, got %d", stats.ActiveConnections)
	}
}

func TestStatsCollector_TotalRequests(t *testing.T) {
	sc := NewStatsCollector([]string{"192.168.1.1"})

	sc.IncTotalRequests()
	sc.IncTotalRequests()
	sc.IncTotalRequests()

	stats := sc.GetStats()
	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
}

func TestStatsCollector_Bytes(t *testing.T) {
	sc := NewStatsCollector([]string{"192.168.1.1"})

	sc.AddBytesSent(1000)
	sc.AddBytesReceived(500)

	stats := sc.GetStats()
	if stats.BytesSent != 1000 {
		t.Errorf("expected 1000 bytes sent, got %d", stats.BytesSent)
	}
	if stats.BytesReceived != 500 {
		t.Errorf("expected 500 bytes received, got %d", stats.BytesReceived)
	}
}

func TestStatsCollector_ConnectionsPerIP(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.1.2"}
	sc := NewStatsCollector(ips)

	sc.IncConnectionsForIP("192.168.1.1")
	sc.IncConnectionsForIP("192.168.1.1")
	sc.IncConnectionsForIP("192.168.1.2")

	stats := sc.GetStats()
	if stats.ConnectionsPerIP["192.168.1.1"] != 2 {
		t.Errorf("expected 2 connections for 192.168.1.1, got %d", stats.ConnectionsPerIP["192.168.1.1"])
	}
	if stats.ConnectionsPerIP["192.168.1.2"] != 1 {
		t.Errorf("expected 1 connection for 192.168.1.2, got %d", stats.ConnectionsPerIP["192.168.1.2"])
	}

	sc.DecConnectionsForIP("192.168.1.1")
	stats = sc.GetStats()
	if stats.ConnectionsPerIP["192.168.1.1"] != 1 {
		t.Errorf("expected 1 connection for 192.168.1.1 after decrement, got %d", stats.ConnectionsPerIP["192.168.1.1"])
	}
}

func TestStatsCollector_SelectionsPerIP(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.1.2"}
	sc := NewStatsCollector(ips)

	sc.IncSelectionsForIP("192.168.1.1", "example.com")
	sc.IncSelectionsForIP("192.168.1.1", "example.com")
	sc.IncSelectionsForIP("192.168.1.2", "other.com")

	stats := sc.GetStats()
	if stats.SelectionsPerIP["192.168.1.1"] != 2 {
		t.Errorf("expected 2 selections for 192.168.1.1, got %d", stats.SelectionsPerIP["192.168.1.1"])
	}
	if stats.SelectionsPerIP["192.168.1.2"] != 1 {
		t.Errorf("expected 1 selection for 192.168.1.2, got %d", stats.SelectionsPerIP["192.168.1.2"])
	}
}

func TestStatsCollector_UnknownIP(t *testing.T) {
	sc := NewStatsCollector([]string{"192.168.1.1"})

	// Should not panic for unknown IP
	sc.IncConnectionsForIP("192.168.1.99")
	sc.DecConnectionsForIP("192.168.1.99")
	sc.IncSelectionsForIP("192.168.1.99", "example.com")
}

func TestStats_Struct(t *testing.T) {
	stats := Stats{
		ActiveConnections: 10,
		TotalRequests:     100,
		BytesSent:         1000,
		BytesReceived:     500,
		ConnectionsPerIP:  map[string]int64{"192.168.1.1": 5},
		SelectionsPerIP:   map[string]int64{"192.168.1.1": 50},
	}

	if stats.ActiveConnections != 10 {
		t.Error("stats struct field mismatch")
	}
}
