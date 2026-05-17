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

type CacheEntry struct {
	Records   []dns.RR
	ExpiresAt time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[CacheKey]CacheEntry
}

func NewCache() *Cache {
	return &Cache{
		entries: make(map[CacheKey]CacheEntry),
	}
}

func (c *Cache) Get(name string, qtype uint16) ([]dns.RR, bool) {
	c.mu.RLock()
	entry, ok := c.entries[CacheKey{Name: name, Type: qtype}]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if !time.Now().Before(entry.ExpiresAt) {
		return nil, false
	}
	result := make([]dns.RR, 0, len(entry.Records))

	for _, rr := range entry.Records {
		copy := rr
		remaining := time.Until(entry.ExpiresAt).Seconds()
		if remaining <= 0 {
			return nil, false
		}
		copy.Header.TTL = uint32(remaining)
		result = append(result, copy)
	}

	return result, true
}

func (c *Cache) Set(name string, qtype uint16, records []dns.RR) {
	if len(records) == 0 {
		return
	}

	c.mu.Lock()
	ttl := minTTL(records)
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
	c.entries[CacheKey{Name: name, Type: qtype}] = CacheEntry{Records: records, ExpiresAt: expiresAt}
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
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
