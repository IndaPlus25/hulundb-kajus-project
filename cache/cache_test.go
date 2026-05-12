package cache

import (
	"hulundb-kajus-dns/dns"
	"net"
	"testing"
)

func TestGetMissingKey(t *testing.T) {
	cache := NewCache()
	records, ok := cache.Get("example.com", 1)

	if ok {
		t.Error("expected ok to be false for missing key")
	}
	if records != nil {
		t.Errorf("expected records to be nil, got %v", records)
	}
}

func TestGetExpiredEntry(t *testing.T) {

	testRecord := dns.RR{
		Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 0},
		Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
	}

	cache := NewCache()

	cache.Set("example.com", 1, []dns.RR{testRecord})

	records, ok := cache.Get("example.com", 1)

	if ok {
		t.Error("expected ok to be false for missing key")
	}
	if records != nil {
		t.Errorf("expected records to be nil, got %v", records)
	}
}

func TestGetLiveEntry(t *testing.T) {

	testRecord := dns.RR{
		Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 300},
		Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
	}

	cache := NewCache()

	cache.Set("example.com", 1, []dns.RR{testRecord})

	records, ok := cache.Get("example.com", 1)

	if !ok {
		t.Error("expected ok to be true for get")
	}
	if records == nil {
		t.Error("expected returned records to non-nil")
	}
}

func TestRoundTrip(t *testing.T) {

	testRecords := []dns.RR{
		{
			Header: dns.RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 300},
			Data:   dns.ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
		},
		{
			Header: dns.RRHeader{Name: "google.com.", Type: 28, Class: 1, TTL: 300},
			Data:   dns.AAAARecord{IP: net.ParseIP("2a00:1450:4005:802::200e")},
		},
		{
			Header: dns.RRHeader{Name: "example.com.", Type: 1, Class: 1, TTL: 300},
			Data:   dns.ARecord{IP: net.IPv4(143, 250, 73, 46).To4()},
		},
		{
			Header: dns.RRHeader{Name: "github.com.", Type: 28, Class: 1, TTL: 300},
			Data:   dns.AAAARecord{IP: net.ParseIP("2a00:1450:4005:802::200e")},
		},
	}

	cache := NewCache()

	cache.Set("example.com", 1, testRecords)

	records, ok := cache.Get("example.com", 1)

	if !ok {
		t.Error("expected ok to be true")
	}
	if len(testRecords) != len(records) {
		t.Errorf("expected %d records, got %d", len(testRecords), len(records))
	}
	for i, expected := range testRecords {
		if expected.Header.Name != records[i].Header.Name ||
			expected.Header.Type != records[i].Header.Type ||
			expected.Header.TTL != records[i].Header.TTL {
			t.Errorf("record %d mismatch: expected %v, got %v", i, expected, records[i])
		}
	}
}
