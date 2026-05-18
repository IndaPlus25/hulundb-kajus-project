package cache

import (
	"hulundb-kajus-dns/dns"
	"sync"
	"time"
)

type CacheKey struct {
	Name string
	Type uint16
}

type entry struct {
	records   []dns.RR
	expiresAt time.Time
	negative  bool
}

type CacheResult struct {
	Records  []dns.RR
	Found    bool
	Negative bool
}

type Cache struct {
	mu      sync.RWMutex
	entries map[CacheKey]entry
}

func NewCache() *Cache {
	return &Cache{
		entries: make(map[CacheKey]entry),
	}
}

func (c *Cache) Get(name string, qtype uint16) CacheResult {
	c.mu.RLock()
	entry, ok := c.entries[CacheKey{Name: name, Type: qtype}]
	c.mu.RUnlock()

	if !ok {
		return CacheResult{nil, false, false}
	}
	if !time.Now().Before(entry.expiresAt) {
		return CacheResult{nil, false, false}
	}
	if entry.negative {
		return CacheResult{nil, true, true}
	}

	result := make([]dns.RR, 0, len(entry.records))

	remaining := time.Until(entry.expiresAt).Seconds()

	for _, rr := range entry.records {
		copy := rr
		copy.Header.TTL = uint32(remaining)
		result = append(result, copy)
	}

	return CacheResult{result, true, false}
}

func (c *Cache) Set(name string, qtype uint16, records []dns.RR) {
	if len(records) == 0 {
		return
	}

	c.mu.Lock()
	ttl := minTTL(records)
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
	c.entries[CacheKey{Name: name, Type: qtype}] = entry{records: records, expiresAt: expiresAt, negative: false}
	c.mu.Unlock()
}

func (c *Cache) SetNegative(name string, qtype uint16, ttl uint32) {
	c.mu.Lock()
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
	c.entries[CacheKey{Name: name, Type: qtype}] = entry{records: nil, expiresAt: expiresAt, negative: true}
	c.mu.Unlock()
}

func minTTL(records []dns.RR) uint32 {
	min := records[0].Header.TTL

	for _, r := range records[1:] {
		if r.Header.TTL < min {
			min = r.Header.TTL
		}
	}
	return min
}

func (c *Cache) StartEviction(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			c.evict()
		}
	}()
}

func (c *Cache) evict() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
