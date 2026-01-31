// Package balancer provides IP load balancing algorithms.
package balancer

import (
	"sync"
	"time"
)

// Entry represents a single usage record.
type Entry struct {
	IP        string
	Timestamp time.Time
}

// HostHistory stores usage history for a single host.
type HostHistory struct {
	entries []Entry
	mu      sync.RWMutex
}

// NewHostHistory creates a new HostHistory.
func NewHostHistory() *HostHistory {
	return &HostHistory{
		entries: make([]Entry, 0, 100),
	}
}

// Add adds an entry to the history.
func (h *HostHistory) Add(ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, Entry{
		IP:        ip,
		Timestamp: time.Now(),
	})
}

// GetFiltered returns entries within the time window and size limit.
func (h *HostHistory) GetFiltered(window time.Duration, maxSize int) []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	result := make([]Entry, 0, maxSize)

	// Start from the end (most recent) and work backwards
	for i := len(h.entries) - 1; i >= 0 && len(result) < maxSize; i-- {
		if h.entries[i].Timestamp.After(cutoff) {
			result = append(result, h.entries[i])
		}
	}

	return result
}

// Cleanup removes expired entries.
func (h *HostHistory) Cleanup(window time.Duration) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().Add(-window)
	newEntries := make([]Entry, 0, len(h.entries))

	for _, e := range h.entries {
		if e.Timestamp.After(cutoff) {
			newEntries = append(newEntries, e)
		}
	}

	removed := len(h.entries) - len(newEntries)
	h.entries = newEntries
	return removed
}

// Len returns the number of entries.
func (h *HostHistory) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

// History stores usage history for all hosts.
type History struct {
	hosts           map[string]*HostHistory
	mu              sync.RWMutex
	maxTotalEntries int // Maximum total entries across all hosts (0 = unlimited)
	totalEntries    int // Current total entry count
}

// HistoryOption is a functional option for History.
type HistoryOption func(*History)

// WithMaxTotalEntries sets the maximum total entries for the history.
func WithMaxTotalEntries(max int) HistoryOption {
	return func(h *History) {
		h.maxTotalEntries = max
	}
}

// NewHistory creates a new History with optional configuration.
func NewHistory(opts ...HistoryOption) *History {
	h := &History{
		hosts: make(map[string]*HostHistory),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// GetOrCreate returns the history for a host, creating it if needed.
func (h *History) GetOrCreate(host string) *HostHistory {
	h.mu.RLock()
	hh, exists := h.hosts[host]
	h.mu.RUnlock()

	if exists {
		return hh
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if hh, exists := h.hosts[host]; exists {
		return hh
	}

	hh = NewHostHistory()
	h.hosts[host] = hh
	return hh
}

// Record records an IP usage for a host.
func (h *History) Record(host, ip string) {
	// Check if we've reached the global limit
	if h.maxTotalEntries > 0 {
		h.mu.Lock()
		if h.totalEntries >= h.maxTotalEntries {
			// Evict oldest entries to make room
			h.evictOldestLocked()
		}
		h.totalEntries++
		h.mu.Unlock()
	}

	hh := h.GetOrCreate(host)
	hh.Add(ip)
}

// evictOldestLocked evicts the oldest entry from any host.
// Must be called with h.mu held.
func (h *History) evictOldestLocked() {
	var oldestHost string
	var oldestTime time.Time
	first := true

	for host, hh := range h.hosts {
		hh.mu.RLock()
		if len(hh.entries) > 0 {
			// First entry is the oldest
			entryTime := hh.entries[0].Timestamp
			if first || entryTime.Before(oldestTime) {
				oldestTime = entryTime
				oldestHost = host
				first = false
			}
		}
		hh.mu.RUnlock()
	}

	if oldestHost != "" {
		hh := h.hosts[oldestHost]
		hh.mu.Lock()
		if len(hh.entries) > 0 {
			hh.entries = hh.entries[1:]
			h.totalEntries--
		}
		// Remove empty host histories
		if len(hh.entries) == 0 {
			delete(h.hosts, oldestHost)
		}
		hh.mu.Unlock()
	}
}

// GetFiltered returns filtered entries for a host.
func (h *History) GetFiltered(host string, window time.Duration, maxSize int) []Entry {
	h.mu.RLock()
	hh, exists := h.hosts[host]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	return hh.GetFiltered(window, maxSize)
}

// Cleanup removes expired entries from all hosts.
func (h *History) Cleanup(window time.Duration) (removedEntries, removedHosts int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	hostsToRemove := make([]string, 0)

	for host, hh := range h.hosts {
		removedEntries += hh.Cleanup(window)
		if hh.Len() == 0 {
			hostsToRemove = append(hostsToRemove, host)
		}
	}

	for _, host := range hostsToRemove {
		delete(h.hosts, host)
		removedHosts++
	}

	return removedEntries, removedHosts
}

// Stats returns history statistics.
func (h *History) Stats() (totalHosts, totalEntries int, entriesPerIP map[string]int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entriesPerIP = make(map[string]int)
	totalHosts = len(h.hosts)

	for _, hh := range h.hosts {
		hh.mu.RLock()
		for _, e := range hh.entries {
			totalEntries++
			entriesPerIP[e.IP]++
		}
		hh.mu.RUnlock()
	}

	return totalHosts, totalEntries, entriesPerIP
}
