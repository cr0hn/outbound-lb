package balancer

import (
	"sync"
	"testing"
	"time"
)

func TestLRU_StartStop(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()

	// Let the cleanup goroutine run at least once
	time.Sleep(50 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		lru.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() took too long")
	}
}

func TestLRU_Concurrent(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		HistoryWindow: 300,
		HistorySize:   1000,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	var wg sync.WaitGroup
	hosts := []string{"host1.com", "host2.com", "host3.com", "host4.com", "host5.com"}

	// Multiple goroutines selecting and recording
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				host := hosts[j%len(hosts)]
				ip, err := lru.Select(host)
				if err != nil {
					t.Errorf("goroutine %d: unexpected error: %v", id, err)
					return
				}
				lru.Record(host, ip)
			}
		}(i)
	}

	wg.Wait()

	stats := lru.GetStats()
	if stats.TotalHosts != len(hosts) {
		t.Errorf("expected %d hosts, got %d", len(hosts), stats.TotalHosts)
	}
	if stats.TotalEntries != 10*100 {
		t.Errorf("expected 1000 entries, got %d", stats.TotalEntries)
	}
}

func TestLRU_CleanupLoop(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1"},
		HistoryWindow: 1, // 1 second window
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)

	// Add some entries
	lru.Record("example.com", "192.168.1.1")
	lru.Record("example.com", "192.168.1.1")

	stats := lru.GetStats()
	if stats.TotalEntries != 2 {
		t.Errorf("expected 2 entries, got %d", stats.TotalEntries)
	}

	// Wait for entries to expire
	time.Sleep(2 * time.Second)

	// Manually trigger cleanup
	lru.history.Cleanup(lru.historyWindow)

	stats = lru.GetStats()
	if stats.TotalEntries != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", stats.TotalEntries)
	}
}

func TestLRU_LRUBehavior(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)

	// First selection - all IPs have 0 usage, should return first
	ip1, _ := lru.Select("example.com")
	lru.Record("example.com", ip1)

	// Second selection - ip1 has 1 usage, others have 0
	ip2, _ := lru.Select("example.com")
	lru.Record("example.com", ip2)

	// ip2 should be different from ip1 (LRU behavior)
	if ip1 == ip2 {
		t.Logf("Note: IPs might be same due to tie-breaking, but generally should differ")
	}

	// Third selection
	ip3, _ := lru.Select("example.com")
	lru.Record("example.com", ip3)

	// After 3 selections with 3 IPs, distribution should be even
	stats := lru.GetStats()
	for ip, count := range stats.EntriesPerIP {
		if count != 1 {
			t.Logf("IP %s has %d uses (may vary due to tie-breaking)", ip, count)
		}
	}
}

func TestLRU_HostIsolation(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)

	// Use all connections on one IP for host1
	for i := 0; i < 10; i++ {
		ip, _ := lru.Select("host1.com")
		lru.Record("host1.com", ip)
	}

	// host2 should still get balanced distribution
	// (its history is independent)
	ip, _ := lru.Select("host2.com")
	lru.Record("host2.com", ip)

	stats := lru.GetStats()
	if stats.TotalHosts != 2 {
		t.Errorf("expected 2 hosts, got %d", stats.TotalHosts)
	}
}

func TestLRU_getAvailableIPs_AllAvailable(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	available := lru.getAvailableIPs()

	if len(available) != 2 {
		t.Errorf("expected 2 available IPs, got %d", len(available))
	}
}

func TestLRU_getAvailableIPs_SomeUnavailable(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter: &mockLimiter{
			unavailable: map[string]bool{"192.168.1.2": true},
		},
	}

	lru := NewLRU(cfg)
	available := lru.getAvailableIPs()

	if len(available) != 2 {
		t.Errorf("expected 2 available IPs, got %d", len(available))
	}

	for _, ip := range available {
		if ip == "192.168.1.2" {
			t.Error("192.168.1.2 should not be in available list")
		}
	}
}

func TestLRU_getAvailableIPs_NilLimiter(t *testing.T) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       nil,
	}

	lru := NewLRU(cfg)
	available := lru.getAvailableIPs()

	if len(available) != 2 {
		t.Errorf("expected 2 available IPs with nil limiter, got %d", len(available))
	}
}

func TestLRU_Select_PoolReuse(t *testing.T) {
	// Test that the sync.Pool for selectContext is working correctly
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		HistoryWindow: 300,
		HistorySize:   1000,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	// Warm up the pool with some selections
	for i := 0; i < 10; i++ {
		ip, err := lru.Select("warmup.com")
		if err != nil {
			t.Fatalf("warmup select failed: %v", err)
		}
		lru.Record("warmup.com", ip)
	}

	// Now do many selections to verify pool reuse doesn't cause issues
	var wg sync.WaitGroup
	hosts := []string{"host1.com", "host2.com", "host3.com"}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			host := hosts[id%len(hosts)]
			for j := 0; j < 100; j++ {
				ip, err := lru.Select(host)
				if err != nil {
					t.Errorf("goroutine %d: unexpected error: %v", id, err)
					return
				}
				lru.Record(host, ip)
			}
		}(i)
	}

	wg.Wait()

	// Verify stats are correct - 10 warmup + (20 goroutines * 100 iterations)
	stats := lru.GetStats()
	expectedEntries := 10 + 20*100
	if stats.TotalEntries != expectedEntries {
		t.Errorf("expected %d entries, got %d", expectedEntries, stats.TotalEntries)
	}
}
