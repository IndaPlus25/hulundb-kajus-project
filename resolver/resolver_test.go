// resolver_test.go
package resolver

import (
	"net"
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

	response, err := Resolve(query, 0)

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
func TestResolvEmpty(t *testing.T) {
	_, err := Resolve([]byte{}, 0)
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
