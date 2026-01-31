package limiter

import (
	"sync"
	"testing"
)

func TestLimiter_Acquire(t *testing.T) {
	l := New(2, 5, []string{"192.168.1.1", "192.168.1.2"})

	// Should succeed
	if err := l.Acquire("192.168.1.1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if l.GetIPCount("192.168.1.1") != 1 {
		t.Error("expected IP count to be 1")
	}

	if l.GetTotalCount() != 1 {
		t.Error("expected total count to be 1")
	}
}

func TestLimiter_Release(t *testing.T) {
	l := New(2, 5, []string{"192.168.1.1"})

	l.Acquire("192.168.1.1")
	l.Release("192.168.1.1")

	if l.GetIPCount("192.168.1.1") != 0 {
		t.Error("expected IP count to be 0 after release")
	}

	if l.GetTotalCount() != 0 {
		t.Error("expected total count to be 0 after release")
	}
}

func TestLimiter_PerIPLimit(t *testing.T) {
	l := New(2, 100, []string{"192.168.1.1"})

	// Should succeed twice
	if err := l.Acquire("192.168.1.1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := l.Acquire("192.168.1.1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Third should fail
	if err := l.Acquire("192.168.1.1"); err != ErrIPLimitReached {
		t.Errorf("expected ErrIPLimitReached, got %v", err)
	}
}

func TestLimiter_TotalLimit(t *testing.T) {
	l := New(10, 3, []string{"192.168.1.1", "192.168.1.2"})

	// Should succeed three times
	l.Acquire("192.168.1.1")
	l.Acquire("192.168.1.1")
	l.Acquire("192.168.1.2")

	// Fourth should fail
	if err := l.Acquire("192.168.1.2"); err != ErrTotalLimitReached {
		t.Errorf("expected ErrTotalLimitReached, got %v", err)
	}
}

func TestLimiter_IsIPAvailable(t *testing.T) {
	l := New(2, 100, []string{"192.168.1.1"})

	if !l.IsIPAvailable("192.168.1.1") {
		t.Error("expected IP to be available")
	}

	l.Acquire("192.168.1.1")
	l.Acquire("192.168.1.1")

	if l.IsIPAvailable("192.168.1.1") {
		t.Error("expected IP to be unavailable after reaching limit")
	}
}

func TestLimiter_GetAvailableIPs(t *testing.T) {
	l := New(1, 100, []string{"192.168.1.1", "192.168.1.2"})

	available := l.GetAvailableIPs([]string{"192.168.1.1", "192.168.1.2"})
	if len(available) != 2 {
		t.Errorf("expected 2 available IPs, got %d", len(available))
	}

	l.Acquire("192.168.1.1")

	available = l.GetAvailableIPs([]string{"192.168.1.1", "192.168.1.2"})
	if len(available) != 1 {
		t.Errorf("expected 1 available IP, got %d", len(available))
	}
}

func TestLimiter_Concurrent(t *testing.T) {
	l := New(100, 1000, []string{"192.168.1.1"})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.Acquire("192.168.1.1"); err == nil {
				l.Release("192.168.1.1")
			}
		}()
	}
	wg.Wait()

	if l.GetTotalCount() != 0 {
		t.Errorf("expected total count to be 0, got %d", l.GetTotalCount())
	}
}

func TestLimiter_StressTest_RaceCondition(t *testing.T) {
	// Stress test to verify CAS-based atomic operations prevent race conditions
	// Run with -race flag to detect data races
	l := New(100, 500, []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})

	const numGoroutines = 1000
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ip := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}[id%3]
			for j := 0; j < iterations; j++ {
				if err := l.Acquire(ip); err == nil {
					// Small delay to increase contention
					l.Release(ip)
				}
			}
		}(i)
	}
	wg.Wait()

	// All connections should be released
	if l.GetTotalCount() != 0 {
		t.Errorf("expected total count to be 0 after stress test, got %d", l.GetTotalCount())
	}

	for _, ip := range []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"} {
		if count := l.GetIPCount(ip); count != 0 {
			t.Errorf("expected IP %s count to be 0, got %d", ip, count)
		}
	}
}

func TestLimiter_StressTest_LimitEnforcement(t *testing.T) {
	// Test that limits are never exceeded even under high contention
	maxPerIP := 10
	maxTotal := 25
	l := New(maxPerIP, maxTotal, []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	maxTotalObserved := int64(0)
	maxPerIPObserved := make(map[string]int64)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ip := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}[id%3]

			for j := 0; j < 50; j++ {
				if err := l.Acquire(ip); err == nil {
					// Check current counts while holding the slot
					totalNow := l.GetTotalCount()
					ipNow := l.GetIPCount(ip)

					mu.Lock()
					if totalNow > maxTotalObserved {
						maxTotalObserved = totalNow
					}
					if ipNow > maxPerIPObserved[ip] {
						maxPerIPObserved[ip] = ipNow
					}
					mu.Unlock()

					l.Release(ip)
				}
			}
		}(i)
	}
	wg.Wait()

	// Verify limits were never exceeded
	if maxTotalObserved > int64(maxTotal) {
		t.Errorf("total limit exceeded: observed %d, max %d", maxTotalObserved, maxTotal)
	}
	for ip, observed := range maxPerIPObserved {
		if observed > int64(maxPerIP) {
			t.Errorf("per-IP limit exceeded for %s: observed %d, max %d", ip, observed, maxPerIP)
		}
	}
}

func TestLimiter_Acquire_PerIPRollback(t *testing.T) {
	// Test that total counter is rolled back when per-IP limit is reached
	l := New(2, 100, []string{"192.168.1.1"})

	// Acquire twice to hit per-IP limit
	l.Acquire("192.168.1.1")
	l.Acquire("192.168.1.1")

	// Third should fail and NOT increment total
	initialTotal := l.GetTotalCount()
	if err := l.Acquire("192.168.1.1"); err != ErrIPLimitReached {
		t.Errorf("expected ErrIPLimitReached, got %v", err)
	}

	// Total should be unchanged
	if l.GetTotalCount() != initialTotal {
		t.Errorf("total count changed after failed per-IP acquire: was %d, now %d",
			initialTotal, l.GetTotalCount())
	}
}

func TestLimiter_Stats(t *testing.T) {
	l := New(10, 100, []string{"192.168.1.1", "192.168.1.2"})

	l.Acquire("192.168.1.1")
	l.Acquire("192.168.1.1")
	l.Acquire("192.168.1.2")

	stats := l.Stats()
	if stats["total"] != 3 {
		t.Errorf("expected total 3, got %d", stats["total"])
	}
	if stats["192.168.1.1"] != 2 {
		t.Errorf("expected 192.168.1.1 count 2, got %d", stats["192.168.1.1"])
	}
	if stats["192.168.1.2"] != 1 {
		t.Errorf("expected 192.168.1.2 count 1, got %d", stats["192.168.1.2"])
	}
}

func TestLimiter_UnknownIP(t *testing.T) {
	l := New(10, 100, []string{"192.168.1.1"})

	// Should handle unknown IP
	if err := l.Acquire("192.168.1.99"); err != nil {
		t.Errorf("unexpected error for unknown IP: %v", err)
	}

	l.Release("192.168.1.99")
}

func TestLimiter_GetAvailableIPs_WithPool(t *testing.T) {
	l := New(1, 100, []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})

	// Get available IPs - should return all 3
	available := l.GetAvailableIPs([]string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})
	if len(available) != 3 {
		t.Errorf("expected 3 available IPs, got %d", len(available))
	}

	// Release slice back to pool
	ReleaseAvailableIPs(available)

	// Acquire one IP
	l.Acquire("192.168.1.1")

	// Get available IPs again - should return 2
	available2 := l.GetAvailableIPs([]string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})
	if len(available2) != 2 {
		t.Errorf("expected 2 available IPs, got %d", len(available2))
	}

	// Verify correct IPs are returned
	hasIP2, hasIP3 := false, false
	for _, ip := range available2 {
		if ip == "192.168.1.2" {
			hasIP2 = true
		}
		if ip == "192.168.1.3" {
			hasIP3 = true
		}
	}
	if !hasIP2 || !hasIP3 {
		t.Error("expected available2 to contain 192.168.1.2 and 192.168.1.3")
	}

	// Release slice
	ReleaseAvailableIPs(available2)
}

func TestReleaseAvailableIPs_LargeSlice(t *testing.T) {
	// Create a large slice that exceeds pool limit
	large := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		large = append(large, "192.168.1.1")
	}

	// Should not panic and should not pool large slices
	ReleaseAvailableIPs(large)

	// Create a small slice that should be pooled
	small := make([]string, 0, 16)
	small = append(small, "192.168.1.1")
	ReleaseAvailableIPs(small)
}

func TestLimiter_GetAvailableIPs_Pool_Concurrent(t *testing.T) {
	l := New(10, 1000, []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				available := l.GetAvailableIPs([]string{"192.168.1.1", "192.168.1.2", "192.168.1.3"})
				if len(available) == 0 {
					continue
				}
				// Use the slice
				_ = available[0]
				// Release back to pool
				ReleaseAvailableIPs(available)
			}
		}()
	}
	wg.Wait()
}
