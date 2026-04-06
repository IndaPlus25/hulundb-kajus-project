package server

import (
	"net"
	"testing"
	"time"
)

// Kriterium 1: Servern startar och lyssnar
func TestServerStarts(t *testing.T) {
	go Start("0.0.0.0:15353")
	time.Sleep(50 * time.Millisecond) // ge servern tid att starta

	// Försök ansluta — om det lyckas lyssnar servern
	conn, err := net.Dial("udp", "127.0.0.1:15353")
	if err != nil {
		t.Fatalf("Server is not listening: %v", err)
	}
	conn.Close()
}

// Kriterium 2: Servern tar emot råa bytes
func TestServerReceivesBytes(t *testing.T) {
	go Start("0.0.0.0:15354")
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("udp", "127.0.0.1:15354")
	if err != nil {
		t.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	// Skicka råa bytes
	payload := []byte{0x00, 0x01, 0x02, 0x03}
	_, err = conn.Write(payload)
	if err != nil {
		t.Fatalf("Could not send bytes: %v", err)
	}
}

// Kriterium 3: Servern kraschar inte vid skräpdata
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

	// Skicka ett paket till — om servern fortfarande svarar har den inte kraschat
	_, err = conn.Write(garbage)
	if err != nil {
		t.Fatalf("Server crashed after garbage input: %v", err)
	}
}
