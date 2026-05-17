package cache

import (
	"hulundb-kajus-dns/dns"
	"net"
	"testing"
	"time"
)

func TestGetMissingKey(t *testing.T) {
	cache := NewCache()
	result := cache.Get("example.com", 1)

	if result.Found {
		t.Error("expected ok to be false for missing key")
	}
	if result.Records != nil {
		t.Errorf("expected records to be nil, got %v", result.Records)
	}
}

func TestGetExpiredEntry(t *testing.T) {

	testRecord := dns.RR{
		Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 0},
		Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
	}

	cache := NewCache()

	cache.Set("example.com", 1, []dns.RR{testRecord})

	result := cache.Get("example.com", 1)

	if result.Found {
		t.Error("expected ok to be false for missing key")
	}
	if result.Records != nil {
		t.Errorf("expected records to be nil, got %v", result.Records)
	}
}

func TestGetLiveEntry(t *testing.T) {

	testRecord := dns.RR{
		Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 300},
		Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
	}

	cache := NewCache()

	cache.Set("example.com", 1, []dns.RR{testRecord})

	result := cache.Get("example.com", 1)

	if !result.Found {
		t.Error("expected ok to be true for get")
	}
	if result.Records == nil {
		t.Error("expected returned records to non-nil")
	}
}

func TestRoundTrip(t *testing.T) {
	testRecords := []dns.RR{
		{
			Header: dns.RRHeader{Name: "example.com.", Type: 1, Class: 1, TTL: 300},
			Data:   dns.ARecord{IP: net.IPv4(143, 250, 73, 46).To4()},
		},
	}

	cache := NewCache()
	cache.Set("example.com", 1, testRecords)

	result := cache.Get("example.com", 1)
	if !result.Found {
		t.Fatal("expected cache hit")
	}
	if len(result.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result.Records))
	}
	if result.Records[0].Header.Name != "example.com." || result.Records[0].Header.Type != 1 {
		t.Errorf("record mismatch: got %v", result.Records[0].Header)
	}
}

// ── Eviction and TTL ────────────────────────────────────────────────────────────────

func TestTTLAdjustment(t *testing.T) {
	testRecords := []dns.RR{
		{
			Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 2},
			Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
		},
	}

	cache := NewCache()

	cache.Set("example.com", 1, testRecords)

	time.Sleep(1 * time.Second)

	result := cache.Get("example.com", 1)

	if !result.Found {
		t.Error("expected ok to be true - unable to get records")
	}
	if result.Records[0].Header.TTL == 0 && result.Records[0].Header.TTL > 1 {
		t.Errorf("expected TTL to be approximately 1, got %d", result.Records[0].Header.TTL)
	}
}

func TestExpiry(t *testing.T) {
	testRecords := []dns.RR{
		{
			Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 1},
			Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
		},
	}

	cache := NewCache()

	cache.Set("example.com", 1, testRecords)

	time.Sleep(1100 * time.Millisecond)

	result := cache.Get("example.com", 1)

	if result.Found {
		t.Error("expected ok to be false - expired TTL expected")
	}
}

func TestBackgroundEviction(t *testing.T) {
	testRecords := []dns.RR{
		{
			Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 1},
			Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
		},
	}

	cache := NewCache()

	cache.Set("example.com", 1, testRecords)

	cache.StartEviction(200 * time.Millisecond)

	time.Sleep(1400 * time.Millisecond)

	if cache.Len() != 0 {
		t.Error("expected empty cache - failed background eviction")
	}
}
