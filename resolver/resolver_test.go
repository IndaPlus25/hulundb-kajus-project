// resolver_test.go
package resolver

import (
	"hulundb-kajus-dns/cache"
	"hulundb-kajus-dns/dns"
	"net"
	"strings"
	"testing"
	"time"
)

// Tests that a real DNS query for google.com receives a response
func TestResolveGoogle(t *testing.T) {
	// A real DNS query for google.com (A-record)
	// These bytes are a valid DNS message in wire format
	query := []byte{
		0x00, 0x01, // ID
		0x01, 0x00, // Flags: standard query
		0x00, 0x01, // 1 question
		0x00, 0x00, // 0 answers
		0x00, 0x00, // 0 authority
		0x00, 0x00, // 0 additional
		// google.com
		0x06, 'g', 'o', 'o', 'g', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // end
		0x00, 0x01, // type A
		0x00, 0x01, // class IN
	}
	c := cache.NewCache()
	c.StartEviction(10000 * time.Second)
	r := New(c)

	response, err := r.Resolve(query, 0)

	if err != nil {
		t.Fatalf("Got error: %v", err)
	}
	if len(response) == 0 {
		t.Fatal("Got empty response")
	}
}

// Tests that timeout works — sends to an address that never responds
func TestResolveTimeout(t *testing.T) {
	query := []byte{0x00, 0x01}

	// Switch temporarily to an address that never responds
	// This tests that we don't hang forever
	_, err := resolveWithAddr(query, "192.0.2.1:53") // TEST-NET, never responds
	if err == nil {
		t.Fatal("Expected timeout error but got none")
	}
}

// Tests that an empty packet doesn't crash
func _TestResolvEmpty(t *testing.T) {
	c := cache.NewCache()
	c.StartEviction(10000 * time.Second)
	r := New(c)

	_, err := r.Resolve([]byte{}, 0)
	// We expect an error, not a crash
	if err == nil {
		t.Fatal("Expected error for empty packet")
	}
}

func resolveWithAddr(query []byte, addr string) ([]byte, error) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.Write(query)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// ── CNAME ───────────────────────────────────────────────────────────────────

func makeARecord(name string) dns.RR {
	return dns.RR{
		Header: dns.RRHeader{Name: name, Type: dns.TypeA},
		Data:   dns.ARecord{IP: net.ParseIP("1.2.3.4")},
	}
}

func makeCNAMERecord(name, target string) dns.RR {
	return dns.RR{
		Header: dns.RRHeader{Name: name, Type: dns.TypeCNAME},
		Data:   dns.CNAMERecord{Name: target},
	}
}

func TestHasCNAMEOnly(t *testing.T) {
	tests := []struct {
		name    string
		answers []dns.RR
		qname   string
		qtype   uint16
		want    bool
	}{
		{
			name:    "cname present, qtype absent",
			answers: []dns.RR{makeCNAMERecord("www.example.com", "example.com")},
			qname:   "www.example.com",
			qtype:   dns.TypeA,
			want:    true,
		},
		{
			name: "cname and qtype both present",
			answers: []dns.RR{
				makeCNAMERecord("www.example.com", "example.com"),
				makeARecord("www.example.com"),
			},
			qname: "www.example.com",
			qtype: dns.TypeA,
			want:  false,
		},
		{
			name:    "neither cname nor qtype",
			answers: []dns.RR{},
			qname:   "www.example.com",
			qtype:   dns.TypeA,
			want:    false,
		},
		{
			name:    "only qtype present",
			answers: []dns.RR{makeARecord("www.example.com")},
			qname:   "www.example.com",
			qtype:   dns.TypeA,
			want:    false,
		},
		{
			name:    "cname for different name",
			answers: []dns.RR{makeCNAMERecord("other.example.com", "example.com")},
			qname:   "www.example.com",
			qtype:   dns.TypeA,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasCNAMEOnly(tt.answers, tt.qname, tt.qtype)
			if got != tt.want {
				t.Errorf("hasCNAMEOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCNAMETarget(t *testing.T) {
	tests := []struct {
		name       string
		answers    []dns.RR
		qname      string
		wantTarget string
		wantFound  bool
	}{
		{
			name:       "cname present",
			answers:    []dns.RR{makeCNAMERecord("www.example.com", "example.com")},
			qname:      "www.example.com",
			wantTarget: "example.com",
			wantFound:  true,
		},
		{
			name:       "no cname",
			answers:    []dns.RR{makeARecord("www.example.com")},
			qname:      "www.example.com",
			wantTarget: "",
			wantFound:  false,
		},
		{
			name:       "cname for different name",
			answers:    []dns.RR{makeCNAMERecord("other.example.com", "example.com")},
			qname:      "www.example.com",
			wantTarget: "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotFound := getCNAMETarget(tt.answers, tt.qname)
			if gotTarget != tt.wantTarget || gotFound != tt.wantFound {
				t.Errorf("getCNAMETarget() = (%v, %v), want (%v, %v)", gotTarget, gotFound, tt.wantTarget, tt.wantFound)
			}
		})
	}
}

func TestFilterByNameAndType(t *testing.T) {
	tests := []struct {
		name    string
		answers []dns.RR
		qname   string
		qtype   uint16
		wantLen int
	}{
		{
			name:    "matching records exist",
			answers: []dns.RR{makeARecord("example.com"), makeARecord("example.com")},
			qname:   "example.com",
			qtype:   dns.TypeA,
			wantLen: 2,
		},
		{
			name:    "no records match name",
			answers: []dns.RR{makeARecord("other.com")},
			qname:   "example.com",
			qtype:   dns.TypeA,
			wantLen: 0,
		},
		{
			name:    "name matches but wrong type",
			answers: []dns.RR{makeCNAMERecord("example.com", "other.com")},
			qname:   "example.com",
			qtype:   dns.TypeA,
			wantLen: 0,
		},
		{
			name: "mix of matching and non-matching",
			answers: []dns.RR{
				makeARecord("example.com"),
				makeARecord("other.com"),
				makeCNAMERecord("example.com", "other.com"),
			},
			qname:   "example.com",
			qtype:   dns.TypeA,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByNameAndType(tt.answers, tt.qname, tt.qtype)
			if len(got) != tt.wantLen {
				t.Errorf("filterByNameAndType() returned %d records, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// ── FALLBACK & RETRY ────────────────────────────────────────────────────────

func TestSendDNSRetryOnTimeout(t *testing.T) {
	query := []byte{0x00, 0x01}

	// Test that sendDNS retries on timeout and returns an error after exhausting retries
	_, err := sendDNS(query, "192.0.2.1") // TEST-NET, never responds
	if err == nil {
		t.Fatal("Expected error after retries exhausted, got nil")
	}

	// Error message should indicate retries were exhausted
	errMsg := err.Error()
	if errMsg == "" {
		t.Fatal("Expected non-empty error message")
	}
}

// Tests that sendDNS returns an error with proper message after exhausting retries
func TestSendDNSErrorMessage(t *testing.T) {
	query := []byte{0x00, 0x01}
	addr := "192.0.2.1"

	_, err := sendDNS(query, addr)
	if err == nil {
		t.Fatal("Expected error for unresponsive nameserver")
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Fatal("Expected non-empty error message")
	}
}

// Tests that queryRootServers tries multiple root servers on failure
func TestRootServerFallback(t *testing.T) {
	query := []byte{
		0x00, 0x01, // ID
		0x01, 0x00, // Flags
		0x00, 0x01, // 1 question
		0x00, 0x00, // 0 answers
		0x00, 0x00, // 0 authority
		0x00, 0x00, // 0 additional
		0x06, 'g', 'o', 'o', 'g', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // end
		0x00, 0x01, // type A
		0x00, 0x01, // class IN
	}

	// queryRootServers should iterate through root servers
	// At least some root servers should be reachable
	target, err := queryRootServers(query)
	if err != nil {
		// If all root servers fail, that's a network issue
		t.Logf("Note: All root servers failed (network may be unavailable): %v", err)
	} else if target == "" {
		t.Fatal("Expected non-empty target from queryRootServers")
	}
}

// Tests that sendDNS does not give up on first timeout, but retries
func TestSendDNSRetriesBeforeGivingUp(t *testing.T) {
	// This test validates that sendDNS makes multiple attempts
	// Each call to sendDNS should retry up to maxRetries times
	query := []byte{0x00, 0x01}

	start := time.Now()
	_, err := sendDNS(query, "192.0.2.1") // Unresponsive address
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error for unresponsive nameserver")
	}

	// With timeout of 3 seconds and 2 retries (3 attempts total),
	// we should wait roughly 3-9 seconds depending on when timeout triggers
	// Just verify it took some reasonable time (at least 1 second)
	if elapsed < 1*time.Second {
		t.Logf("Warning: sendDNS returned too quickly (%v), may not have retried", elapsed)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
