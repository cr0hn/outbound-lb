package limiter

import (
	"fmt"
	"testing"
)

func BenchmarkLimiter_AcquireRelease(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(1000, 10000, ips)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := ips[i%len(ips)]
		if err := lim.Acquire(ip); err == nil {
			lim.Release(ip)
		}
	}
}

func BenchmarkLimiter_AcquireRelease_Parallel(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100000, 1000000, ips)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := ips[i%len(ips)]
			if err := lim.Acquire(ip); err == nil {
				lim.Release(ip)
			}
			i++
		}
	})
}

func BenchmarkLimiter_AcquireRelease_HighContention(b *testing.B) {
	// Single IP - maximum contention
	ips := []string{"192.168.1.1"}
	lim := New(100000, 1000000, ips)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := lim.Acquire("192.168.1.1"); err == nil {
				lim.Release("192.168.1.1")
			}
		}
	})
}

func BenchmarkLimiter_IsIPAvailable(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100, 1000, ips)

	// Acquire some connections
	for i := 0; i < 50; i++ {
		lim.Acquire(ips[i%len(ips)])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lim.IsIPAvailable(ips[i%len(ips)])
	}
}

func BenchmarkLimiter_IsIPAvailable_Parallel(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100, 1000, ips)

	// Acquire some connections
	for i := 0; i < 50; i++ {
		lim.Acquire(ips[i%len(ips)])
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = lim.IsIPAvailable(ips[i%len(ips)])
			i++
		}
	})
}

func BenchmarkLimiter_GetAvailableIPs(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.5", "192.168.1.6", "192.168.1.7", "192.168.1.8"}
	lim := New(100, 1000, ips)

	// Make half unavailable
	for i := 0; i < 4; i++ {
		for j := 0; j < 100; j++ {
			lim.Acquire(ips[i])
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		available := lim.GetAvailableIPs(ips)
		ReleaseAvailableIPs(available)
	}
}

func BenchmarkLimiter_GetAvailableIPs_Parallel(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4", "192.168.1.5", "192.168.1.6", "192.168.1.7", "192.168.1.8"}
	lim := New(100, 1000, ips)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			available := lim.GetAvailableIPs(ips)
			ReleaseAvailableIPs(available)
		}
	})
}

func BenchmarkLimiter_GetIPCount(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100, 1000, ips)

	// Acquire some connections
	for i := 0; i < 50; i++ {
		lim.Acquire(ips[i%len(ips)])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lim.GetIPCount(ips[i%len(ips)])
	}
}

func BenchmarkLimiter_GetTotalCount(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100, 1000, ips)

	// Acquire some connections
	for i := 0; i < 50; i++ {
		lim.Acquire(ips[i%len(ips)])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lim.GetTotalCount()
	}
}

func BenchmarkLimiter_Stats(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100, 1000, ips)

	// Acquire some connections
	for i := 0; i < 50; i++ {
		lim.Acquire(ips[i%len(ips)])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lim.Stats()
	}
}

func BenchmarkLimiter_UpdateLimits(b *testing.B) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	lim := New(100, 1000, ips)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lim.UpdateLimits(100+i%10, 1000+i%100)
	}
}

func BenchmarkAvailableIPsPool(b *testing.B) {
	// Test the pool efficiency
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := availableIPsPool.Get().([]string)
			s = s[:0]
			s = append(s, "test1", "test2", "test3")
			ReleaseAvailableIPs(s)
		}
	})
}

// Test scaling with number of IPs
func BenchmarkLimiter_ScalingIPs(b *testing.B) {
	for _, numIPs := range []int{4, 8, 16, 32, 64} {
		b.Run(fmt.Sprintf("IPs_%d", numIPs), func(b *testing.B) {
			ips := make([]string, numIPs)
			for i := range ips {
				ips[i] = fmt.Sprintf("192.168.%d.%d", i/256, i%256)
			}
			lim := New(100, numIPs*100, ips)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					ip := ips[i%len(ips)]
					if err := lim.Acquire(ip); err == nil {
						lim.Release(ip)
					}
					i++
				}
			})
		})
	}
}
