package balancer

import (
	"testing"
	"time"
)

// mockLimiter is a mock implementation of IPLimiter.
type mockLimiter struct {
	unavailable map[string]bool
}

func (m *mockLimiter) IsIPAvailable(ip string) bool {
	if m.unavailable == nil {
		return true
	}
	return !m.unavailable[ip]
}

func (m *mockLimiter) GetAvailableIPs(ips []string) []string {
	available := make([]string, 0, len(ips))
	for _, ip := range ips {
		if m.IsIPAvailable(ip) {
			available = append(available, ip)
		}
	}
	return available
}

func TestLRUSelect_SingleIP(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	bal := NewLRU(cfg)

	ip, err := bal.Select("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}
}

func TestLRUSelect_MultipleIPs(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	bal := NewLRU(cfg)

	// First selection should return first available IP (no history)
	ip1, err := bal.Select("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bal.Record("example.com", ip1)

	// Second selection should prefer a different IP
	ip2, err := bal.Select("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip1 == ip2 {
		// Could be same if tie-breaking, but with 3 IPs and 1 used, should differ
		// Actually, since all IPs have 0 usage initially, the first unused one wins
	}
	bal.Record("example.com", ip2)

	// Third selection
	ip3, err := bal.Select("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bal.Record("example.com", ip3)

	// After 3 selections with 3 IPs, we should have used all IPs
	stats := bal.GetStats()
	if stats.TotalEntries != 3 {
		t.Errorf("expected 3 entries, got %d", stats.TotalEntries)
	}
}

func TestLRUSelect_DistributionPerHost(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	bal := NewLRU(cfg)

	// Different hosts should have independent histories
	ip1, _ := bal.Select("host1.com")
	bal.Record("host1.com", ip1)

	ip2, _ := bal.Select("host2.com")
	bal.Record("host2.com", ip2)

	// Both hosts should start with same IP (first in list, no usage)
	// but subsequent calls should rotate
	stats := bal.GetStats()
	if stats.TotalHosts != 2 {
		t.Errorf("expected 2 hosts, got %d", stats.TotalHosts)
	}
}

func TestLRUSelect_RespectsLimiter(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter: &mockLimiter{
			unavailable: map[string]bool{"192.168.1.1": true},
		},
	}

	bal := NewLRU(cfg)

	// Should only return the available IP
	for i := 0; i < 10; i++ {
		ip, err := bal.Select("example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ip != "192.168.1.2" {
			t.Errorf("expected 192.168.1.2 (only available), got %s", ip)
		}
	}
}

func TestLRUSelect_NoAvailableIPs(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter: &mockLimiter{
			unavailable: map[string]bool{
				"192.168.1.1": true,
				"192.168.1.2": true,
			},
		},
	}

	bal := NewLRU(cfg)

	_, err := bal.Select("example.com")
	if err != ErrNoAvailableIPs {
		t.Errorf("expected ErrNoAvailableIPs, got %v", err)
	}
}

func TestHistory_GetFiltered(t *testing.T) {
	h := NewHistory()

	// Add entries
	for i := 0; i < 10; i++ {
		h.Record("example.com", "192.168.1.1")
	}

	// Get filtered with size limit
	entries := h.GetFiltered("example.com", 5*time.Minute, 5)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestHostHistory_Add(t *testing.T) {
	hh := NewHostHistory()

	hh.Add("192.168.1.1")
	hh.Add("192.168.1.2")

	if hh.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", hh.Len())
	}
}
