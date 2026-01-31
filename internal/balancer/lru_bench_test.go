package balancer

import (
	"fmt"
	"sync"
	"testing"
)

func BenchmarkLRU_Select(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip, _ := lru.Select("example.com")
		lru.Record("example.com", ip)
	}
}

func BenchmarkLRU_Select_MultipleHosts(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	hosts := make([]string, 100)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("host%d.example.com", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		host := hosts[i%len(hosts)]
		ip, _ := lru.Select(host)
		lru.Record(host, ip)
	}
}

func BenchmarkLRU_Select_Parallel(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		HistoryWindow: 300,
		HistorySize:   1000,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	hosts := make([]string, 100)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("host%d.example.com", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			host := hosts[i%len(hosts)]
			ip, _ := lru.Select(host)
			lru.Record(host, ip)
			i++
		}
	})
}

func BenchmarkLRU_Select_HighContention(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2"},
		HistoryWindow: 300,
		HistorySize:   1000,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	// All goroutines hitting the same host - maximum contention
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ip, _ := lru.Select("contention.example.com")
			lru.Record("contention.example.com", ip)
		}
	})
}

func BenchmarkHistory_Record(b *testing.B) {
	history := NewHistory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history.Record("example.com", "192.168.1.1")
	}
}

func BenchmarkHistory_Record_MultipleHosts(b *testing.B) {
	history := NewHistory()
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

func BenchmarkHistory_Record_Parallel(b *testing.B) {
	history := NewHistory()
	hosts := make([]string, 100)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("host%d.example.com", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			host := hosts[i%len(hosts)]
			history.Record(host, "192.168.1.1")
			i++
		}
	})
}

func BenchmarkHistory_GetFiltered(b *testing.B) {
	history := NewHistory()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		history.Record("example.com", "192.168.1.1")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		history.GetFiltered("example.com", 300, 100)
	}
}

func BenchmarkHistory_WithBounds(b *testing.B) {
	history := NewHistory(WithMaxTotalEntries(10000))
	hosts := make([]string, 100)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("host%d.example.com", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		host := hosts[i%len(hosts)]
		history.Record(host, "192.168.1.1")
	}
}

func BenchmarkSelectContextPool(b *testing.B) {
	// Test the selectContext pool efficiency
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := selectContextPool.Get().(*selectContext)
			ctx.usageCount["test"] = 1
			ctx.lastUsed["test"] = ctx.lastUsed["test"] // Simulate access
			clear(ctx.usageCount)
			clear(ctx.lastUsed)
			selectContextPool.Put(ctx)
		}
	})
}

func BenchmarkLRU_GetAvailableIPs(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.5", "192.168.1.6", "192.168.1.7", "192.168.1.8"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lru.getAvailableIPs()
	}
}

func BenchmarkLRU_GetAvailableIPs_Parallel(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.5", "192.168.1.6", "192.168.1.7", "192.168.1.8"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = lru.getAvailableIPs()
		}
	})
}

// Benchmark to compare allocation patterns
func BenchmarkLRU_SelectAllocationTest(b *testing.B) {
	cfg := Config{
		IPs:           []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		HistoryWindow: 300,
		HistorySize:   100,
		Limiter:       &mockLimiter{},
	}

	lru := NewLRU(cfg)
	lru.Start()
	defer lru.Stop()

	b.ReportAllocs()
	b.ResetTimer()

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < b.N/10; j++ {
				ip, _ := lru.Select(fmt.Sprintf("host%d.com", id))
				mu.Lock()
				lru.Record(fmt.Sprintf("host%d.com", id), ip)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
}
