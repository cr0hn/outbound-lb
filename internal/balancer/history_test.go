package balancer

import (
	"sync"
	"testing"
	"time"
)

func TestNewHostHistory(t *testing.T) {
	hh := NewHostHistory()
	if hh == nil {
		t.Fatal("expected non-nil host history")
	}
	if hh.Len() != 0 {
		t.Errorf("expected empty history, got %d entries", hh.Len())
	}
}

func TestHostHistory_AddAndLen(t *testing.T) {
	hh := NewHostHistory()

	hh.Add("192.168.1.1")
	if hh.Len() != 1 {
		t.Errorf("expected 1 entry, got %d", hh.Len())
	}

	hh.Add("192.168.1.2")
	hh.Add("192.168.1.3")
	if hh.Len() != 3 {
		t.Errorf("expected 3 entries, got %d", hh.Len())
	}
}

func TestHostHistory_GetFiltered_SizeLimit(t *testing.T) {
	hh := NewHostHistory()

	// Add 20 entries
	for i := 0; i < 20; i++ {
		hh.Add("192.168.1.1")
	}

	// Get with size limit of 5
	entries := hh.GetFiltered(time.Hour, 5)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestHostHistory_GetFiltered_TimeWindow(t *testing.T) {
	hh := NewHostHistory()

	// Add entry
	hh.Add("192.168.1.1")

	// Get with very small window (entries should be filtered)
	time.Sleep(10 * time.Millisecond)
	entries := hh.GetFiltered(1*time.Nanosecond, 100)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries with small window, got %d", len(entries))
	}

	// Get with large window (entries should be included)
	entries = hh.GetFiltered(time.Hour, 100)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry with large window, got %d", len(entries))
	}
}

func TestHostHistory_Cleanup(t *testing.T) {
	hh := NewHostHistory()

	// Add entries
	hh.Add("192.168.1.1")
	hh.Add("192.168.1.2")
	hh.Add("192.168.1.3")

	if hh.Len() != 3 {
		t.Fatalf("expected 3 entries before cleanup, got %d", hh.Len())
	}

	// Wait and cleanup with small window
	time.Sleep(10 * time.Millisecond)
	removed := hh.Cleanup(1 * time.Nanosecond)

	if removed != 3 {
		t.Errorf("expected 3 removed, got %d", removed)
	}
	if hh.Len() != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", hh.Len())
	}
}

func TestHostHistory_Cleanup_KeepRecent(t *testing.T) {
	hh := NewHostHistory()

	// Add entry
	hh.Add("192.168.1.1")

	// Cleanup with large window (should keep entry)
	removed := hh.Cleanup(time.Hour)

	if removed != 0 {
		t.Errorf("expected 0 removed, got %d", removed)
	}
	if hh.Len() != 1 {
		t.Errorf("expected 1 entry after cleanup, got %d", hh.Len())
	}
}

func TestHostHistory_Concurrent(t *testing.T) {
	hh := NewHostHistory()
	var wg sync.WaitGroup

	// Multiple writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				hh.Add("192.168.1.1")
			}
		}()
	}

	// Multiple readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = hh.GetFiltered(time.Hour, 50)
				_ = hh.Len()
			}
		}()
	}

	wg.Wait()

	if hh.Len() != 1000 {
		t.Errorf("expected 1000 entries, got %d", hh.Len())
	}
}

func TestNewHistory(t *testing.T) {
	h := NewHistory()
	if h == nil {
		t.Fatal("expected non-nil history")
	}
	if len(h.hosts) != 0 {
		t.Errorf("expected empty hosts map, got %d entries", len(h.hosts))
	}
}

func TestHistory_GetOrCreate(t *testing.T) {
	h := NewHistory()

	hh1 := h.GetOrCreate("host1.com")
	if hh1 == nil {
		t.Fatal("expected non-nil host history")
	}

	// Get same host again
	hh2 := h.GetOrCreate("host1.com")
	if hh1 != hh2 {
		t.Error("expected same host history instance")
	}

	// Get different host
	hh3 := h.GetOrCreate("host2.com")
	if hh1 == hh3 {
		t.Error("expected different host history for different host")
	}
}

func TestHistory_Record(t *testing.T) {
	h := NewHistory()

	h.Record("host1.com", "192.168.1.1")
	h.Record("host1.com", "192.168.1.2")
	h.Record("host2.com", "192.168.1.1")

	entries1 := h.GetFiltered("host1.com", time.Hour, 100)
	if len(entries1) != 2 {
		t.Errorf("expected 2 entries for host1.com, got %d", len(entries1))
	}

	entries2 := h.GetFiltered("host2.com", time.Hour, 100)
	if len(entries2) != 1 {
		t.Errorf("expected 1 entry for host2.com, got %d", len(entries2))
	}
}

func TestHistory_GetFiltered_NonexistentHost(t *testing.T) {
	h := NewHistory()

	entries := h.GetFiltered("nonexistent.com", time.Hour, 100)
	if entries != nil {
		t.Errorf("expected nil for nonexistent host, got %v", entries)
	}
}

func TestHistory_Cleanup(t *testing.T) {
	h := NewHistory()

	h.Record("host1.com", "192.168.1.1")
	h.Record("host2.com", "192.168.1.2")

	// Wait and cleanup
	time.Sleep(10 * time.Millisecond)
	removedEntries, removedHosts := h.Cleanup(1 * time.Nanosecond)

	if removedEntries != 2 {
		t.Errorf("expected 2 removed entries, got %d", removedEntries)
	}
	if removedHosts != 2 {
		t.Errorf("expected 2 removed hosts, got %d", removedHosts)
	}
}

func TestHistory_Stats(t *testing.T) {
	h := NewHistory()

	h.Record("host1.com", "192.168.1.1")
	h.Record("host1.com", "192.168.1.1")
	h.Record("host1.com", "192.168.1.2")
	h.Record("host2.com", "192.168.1.1")

	totalHosts, totalEntries, entriesPerIP := h.Stats()

	if totalHosts != 2 {
		t.Errorf("expected 2 hosts, got %d", totalHosts)
	}
	if totalEntries != 4 {
		t.Errorf("expected 4 entries, got %d", totalEntries)
	}
	if entriesPerIP["192.168.1.1"] != 3 {
		t.Errorf("expected 3 entries for 192.168.1.1, got %d", entriesPerIP["192.168.1.1"])
	}
	if entriesPerIP["192.168.1.2"] != 1 {
		t.Errorf("expected 1 entry for 192.168.1.2, got %d", entriesPerIP["192.168.1.2"])
	}
}

func TestHistory_Concurrent(t *testing.T) {
	h := NewHistory()
	var wg sync.WaitGroup
	hosts := []string{"host1.com", "host2.com", "host3.com"}
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	// Multiple writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				host := hosts[j%len(hosts)]
				ip := ips[j%len(ips)]
				h.Record(host, ip)
			}
		}(i)
	}

	// Multiple readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				for _, host := range hosts {
					_ = h.GetFiltered(host, time.Hour, 50)
				}
				_, _, _ = h.Stats()
			}
		}()
	}

	wg.Wait()
}

func TestEntry_Fields(t *testing.T) {
	now := time.Now()
	e := Entry{
		IP:        "192.168.1.1",
		Timestamp: now,
	}

	if e.IP != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", e.IP)
	}
	if e.Timestamp != now {
		t.Errorf("expected timestamp %v, got %v", now, e.Timestamp)
	}
}
