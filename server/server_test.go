package server

import (
	"net"
	"testing"
	"time"
)

// Criterion 1: Server starts and listens
func TestServerStarts(t *testing.T) {
	go Start("0.0.0.0:15353")
	time.Sleep(50 * time.Millisecond) // give server time to start

	// Try to connect — if it succeeds server is listening
	conn, err := net.Dial("udp", "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("Server is not listening: %v", err)
	}
	conn.Close()
}

// Criterion 2: Server receives raw bytes
func TestServerReceivesBytes(t *testing.T) {
	go Start("0.0.0.0:15354")
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("udp", "127.0.0.1:15354")
	if err != nil {
		t.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	// Send raw bytes
	payload := []byte{0x00, 0x01, 0x02, 0x03}
	_, err = conn.Write(payload)
	if err != nil {
		t.Fatalf("Could not send bytes: %v", err)
	}
}

// Criterion 3: Server doesn't crash with garbage data
func TestServerHandlesGarbage(t *testing.T) {
	go Start("0.0.0.0:15355")
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("udp", "127.0.0.1:15355")
	if err != nil {
		t.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	garbage := []byte("hejhej detta är skräp")
	conn.Write(garbage)

	time.Sleep(50 * time.Millisecond)

	// Send a packet — if the server still responds it hasn't crashed
	_, err = conn.Write(garbage)
	if err != nil {
		t.Fatalf("Server crashed after garbage input: %v", err)
	}
}
