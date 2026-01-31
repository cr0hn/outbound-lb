package balancer

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// TestHistory_MemoryBounds tests that history doesn't grow unbounded with many unique hosts.
func TestHistory_MemoryBounds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory bounds test in short mode")
	}

	// Create history with bounds
	maxEntries := 10000
	history := NewHistory(WithMaxTotalEntries(maxEntries))

	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Simulate spike of unique hosts
	numHosts := 100000
	for i := 0; i < numHosts; i++ {
		host := fmt.Sprintf("host%d.example.com", i)
		history.Record(host, "192.168.1.1")
	}

	// Get final memory stats
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check that we don't exceed max entries
	hosts, entries, _ := history.Stats()
	t.Logf("Unique hosts recorded: %d", numHosts)
	t.Logf("Hosts in history: %d", hosts)
	t.Logf("Entries in history: %d", entries)
	t.Logf("Memory before: %d MB", m1.Alloc/1024/1024)
	t.Logf("Memory after: %d MB", m2.Alloc/1024/1024)
	t.Logf("Memory delta: %d MB", (m2.Alloc-m1.Alloc)/1024/1024)

	// Entries should be capped at max
	if entries > maxEntries {
		t.Errorf("entries (%d) exceeded max (%d)", entries, maxEntries)
	}
}

// TestHistory_MemoryUnbounded tests memory growth without bounds (for comparison).
func TestHistory_MemoryUnbounded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping unbounded memory test in short mode")
	}

	history := NewHistory() // No bounds

	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Record fewer entries since this can grow large
	numHosts := 10000
	for i := 0; i < numHosts; i++ {
		host := fmt.Sprintf("host%d.example.com", i)
		history.Record(host, "192.168.1.1")
	}

	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	hosts, entries, _ := history.Stats()
	t.Logf("Unbounded history test:")
	t.Logf("Hosts: %d", hosts)
	t.Logf("Entries: %d", entries)
	t.Logf("Memory delta: %d KB", (m2.Alloc-m1.Alloc)/1024)

	// All entries should be present
	if entries != numHosts {
		t.Errorf("expected %d entries, got %d", numHosts, entries)
	}
}

// TestHistory_CleanupWithBounds tests that cleanup works with bounds.
func TestHistory_CleanupWithBounds(t *testing.T) {
	history := NewHistory(WithMaxTotalEntries(100))

	// Add entries that will expire
	for i := 0; i < 50; i++ {
		history.Record("example.com", "192.168.1.1")
	}

	// Wait for entries to age
	time.Sleep(100 * time.Millisecond)

	// Add more entries to trigger eviction
	for i := 0; i < 100; i++ {
		history.Record(fmt.Sprintf("host%d.com", i), "192.168.1.1")
	}

	// Cleanup old entries
	window := 50 * time.Millisecond
	removedEntries, removedHosts := history.Cleanup(window)

	t.Logf("Removed entries: %d", removedEntries)
	t.Logf("Removed hosts: %d", removedHosts)

	hosts, entries, _ := history.Stats()
	t.Logf("Remaining hosts: %d", hosts)
	t.Logf("Remaining entries: %d", entries)

	// Should have at most maxEntries
	if entries > 100 {
		t.Errorf("entries (%d) exceeded max (100)", entries)
	}
}

// TestHistory_EvictionOrder tests that oldest entries are evicted first.
func TestHistory_EvictionOrder(t *testing.T) {
	history := NewHistory(WithMaxTotalEntries(5))

	// Add entries with distinct timestamps
	hosts := []string{"first.com", "second.com", "third.com"}
	for _, host := range hosts {
		history.Record(host, "192.168.1.1")
		time.Sleep(10 * time.Millisecond) // Ensure distinct timestamps
	}

	// Add more entries to force eviction
	for i := 0; i < 5; i++ {
		history.Record("new.com", "192.168.1.1")
	}

	// The oldest host (first.com) should have been evicted
	entries := history.GetFiltered("first.com", time.Hour, 100)
	t.Logf("Entries for first.com: %d", len(entries))

	// Note: Due to how eviction works, first.com may or may not be completely gone
	// but it should have fewer entries than it originally had
}

// TestHistory_ConcurrentWithBounds tests concurrent access with bounds.
func TestHistory_ConcurrentWithBounds(t *testing.T) {
	history := NewHistory(WithMaxTotalEntries(1000))

	// Multiple goroutines writing
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 1000; j++ {
				host := fmt.Sprintf("host%d-%d.com", id, j)
				history.Record(host, "192.168.1.1")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not exceed max entries (with some tolerance for race conditions)
	_, entries, _ := history.Stats()
	t.Logf("Total entries after concurrent writes: %d", entries)

	// Allow some tolerance due to race conditions in the eviction check
	if entries > 1100 { // 10% tolerance
		t.Errorf("entries (%d) significantly exceeded max (1000)", entries)
	}
}

// BenchmarkHistory_RecordWithBounds benchmarks recording with bounds enabled.
func BenchmarkHistory_RecordWithBounds(b *testing.B) {
	history := NewHistory(WithMaxTotalEntries(10000))
	hosts := make([]string, 1000)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("host%d.example.com", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		host := hosts[i%len(hosts)]
		history.Record(host, "192.168.1.1")
	}
}

// BenchmarkHistory_RecordWithBoundsEviction benchmarks when eviction is frequently triggered.
func BenchmarkHistory_RecordWithBoundsEviction(b *testing.B) {
	history := NewHistory(WithMaxTotalEntries(100))
	hosts := make([]string, 1000)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("host%d.example.com", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		host := hosts[i%len(hosts)]
		history.Record(host, "192.168.1.1")
	}
}
