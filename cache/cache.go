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

	if !time.Now().Before(entry.ExpiresAt) {
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return entry.Records, true
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
