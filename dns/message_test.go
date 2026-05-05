package dns

import (
	"net"
	"testing"
)

// ── EncodeName ────────────────────────────────────────────────────────────────

func TestEncodeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:  "single label with trailing dot",
			input: "com.",
			want:  []byte{3, 'c', 'o', 'm', 0},
		},
		{
			name:  "multi label with trailing dot",
			input: "www.example.com.",
			want:  []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
		},
		{
			name:  "multi label without trailing dot",
			input: "www.example.com",
			want:  []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
		},
		{
			name:  "root label",
			input: ".",
			want:  []byte{0},
		},
		{
			name:  "single character labels",
			input: "a.b.",
			want:  []byte{1, 'a', 1, 'b', 0},
		},
		{
			name:  "max length label (63 chars)",
			input: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.",
			want: append(
				append([]byte{63}, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")...),
				0,
			),
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "label too long (64 chars)",
			input:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.",
			wantErr: true,
		},
		{
			name:    "empty label in middle",
			input:   "www..example.com.",
			wantErr: true,
		},
		{
			name: "name too long",
			// 8 labels of 30 chars each = 8*(30+1) + 1 = 249 bytes... nudge over 255
			// 9 labels of 29 chars: 9*(29+1)+1 = 271 > 255
			input:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.aaaaaaaaaaaaaaaaaaaaaaaaaaaaa.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("EncodeName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && !bytesEqual(got, tt.want) {
				t.Errorf("EncodeName(%q)\n got  %v\n want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ── DecodeName ────────────────────────────────────────────────────────────────

func TestDecodeName(t *testing.T) {
	tests := []struct {
		name         string
		msg          []byte
		offset       int
		wantName     string
		wantConsumed int
		wantErr      bool
	}{
		{
			name:         "single label",
			msg:          []byte{3, 'c', 'o', 'm', 0},
			offset:       0,
			wantName:     "com.",
			wantConsumed: 5,
		},
		{
			name:         "multi label",
			msg:          []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			offset:       0,
			wantName:     "www.example.com.",
			wantConsumed: 17,
		},
		{
			name:         "root label",
			msg:          []byte{0},
			offset:       0,
			wantName:     ".",
			wantConsumed: 1,
		},
		{
			name: "name not at offset zero",
			msg: []byte{
				0xFF, 0xFF, 0xFF, // junk
				3, 'c', 'o', 'm', 0,
			},
			offset:       3,
			wantName:     "com.",
			wantConsumed: 5,
		},
		{
			// "example.com." at offset 0, then "www" + pointer to 0 at offset 13
			name: "single pointer",
			msg: []byte{
				7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
				3, 'c', 'o', 'm',
				0,
				3, 'w', 'w', 'w',
				0xC0, 0x00,
			},
			offset:       13,
			wantName:     "www.example.com.",
			wantConsumed: 6, // "www" label (4 bytes) + 2 pointer bytes
		},
		{
			// "com." at offset 0
			// "example" + pointer to 0 at offset 5
			// "www" + pointer to 5 at offset 14
			name: "pointer to pointer chain",
			msg: []byte{
				3, 'c', 'o', 'm', 0, // offset 0: "com."
				7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', // offset 5
				0xC0, 0x00, // offset 13: → 0
				3, 'w', 'w', 'w', // offset 15
				0xC0, 0x05, // offset 19: → 5
			},
			offset:       15,
			wantName:     "www.example.com.",
			wantConsumed: 6,
		},
		{
			name: "pointer at start of name",
			msg: []byte{
				3, 'c', 'o', 'm', 0, // offset 0: "com."
				0xC0, 0x00, // offset 5: pointer straight to 0
			},
			offset:       5,
			wantName:     "com.",
			wantConsumed: 2,
		},
		{
			name: "direct pointer loop",
			msg: []byte{
				0xC0, 0x02,
				0xC0, 0x00,
			},
			offset:  0,
			wantErr: true,
		},
		{
			name: "indirect pointer loop (A→B→C→A)",
			msg: []byte{
				0xC0, 0x02, // offset 0 → 2
				0xC0, 0x04, // offset 2 → 4
				0xC0, 0x00, // offset 4 → 0
			},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "pointer target out of bounds",
			msg:     []byte{0xC0, 0xFF},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "truncated label",
			msg:     []byte{5, 'a', 'b'}, // says 5 bytes, only 2 follow
			offset:  0,
			wantErr: true,
		},
		{
			name:    "truncated pointer — missing second byte",
			msg:     []byte{0xC0},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "reserved label type 0x40",
			msg:     []byte{0x40, 'a', 'b', 'c', 0},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "reserved label type 0x80",
			msg:     []byte{0x80, 'a', 'b', 'c', 0},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "label length exceeds max (64)",
			msg:     append([]byte{64}, make([]byte, 65)...),
			offset:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotConsumed, err := DecodeName(tt.msg, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotName != tt.wantName {
				t.Errorf("name: got %q, want %q", gotName, tt.wantName)
			}
			if gotConsumed != tt.wantConsumed {
				t.Errorf("consumed: got %d, want %d", gotConsumed, tt.wantConsumed)
			}
		})
	}
}

// ── Round-trip ────────────────────────────────────────────────────────────────

func TestEncodeDecodeRoundTrip(t *testing.T) {
	cases := []string{
		"com.",
		"www.example.com.",
		"www.example.com", // no trailing dot — canonical form after round-trip should be with dot
		".",
		"a.b.",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.", // 63-char label
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			encoded, err := EncodeName(input)
			if err != nil {
				t.Fatalf("EncodeName(%q) unexpected error: %v", input, err)
			}
			decoded, consumed, err := DecodeName(encoded, 0)
			if err != nil {
				t.Fatalf("DecodeName() unexpected error: %v", err)
			}
			if consumed != len(encoded) {
				t.Errorf("consumed %d bytes, encoded length was %d", consumed, len(encoded))
			}
			// normalise input for comparison: ensure trailing dot
			want := input
			if len(want) == 0 || want[len(want)-1] != '.' {
				want += "."
			}
			if decoded != want {
				t.Errorf("round-trip: got %q, want %q", decoded, want)
			}
		})
	}
}

// ── Fuzz ──────────────────────────────────────────────────────────────────────

func FuzzDecodeName(f *testing.F) {
	f.Add([]byte{3, 'w', 'w', 'w', 0}, 0)
	f.Add([]byte{0}, 0)
	f.Add([]byte{
		7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		3, 'c', 'o', 'm', 0,
		3, 'w', 'w', 'w', 0xC0, 0x00,
	}, 13)
	f.Add([]byte{0xC0, 0x00}, 0) // pointer loop
	f.Add([]byte{}, 0)

	f.Fuzz(func(t *testing.T, msg []byte, offset int) {
		if offset < 0 || offset >= len(msg) {
			return
		}
		name, consumed, err := DecodeName(msg, offset)
		if err != nil {
			return
		}
		if consumed <= 0 || consumed > len(msg)-offset {
			t.Errorf("invalid consumed=%d for offset=%d len=%d", consumed, offset, len(msg))
		}
		if len(name) == 0 || name[len(name)-1] != '.' {
			t.Errorf("name %q does not end with '.'", name)
		}
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ── Header ────────────────────────────────────────────────────────────────
func TestHeaderRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		header Header
	}{
		{
			name: "query header",
			header: Header{
				ID:      0x1234,
				QR:      false,
				Opcode:  OpcodeQuery,
				RD:      true,
				QDCount: 1,
			},
		},
		{
			name: "response header",
			header: Header{
				ID:      0x1234,
				QR:      true,
				Opcode:  OpcodeQuery,
				AA:      true,
				RD:      true,
				RA:      true,
				RCode:   RCodeNoError,
				QDCount: 1,
				ANCount: 1,
			},
		},
		{
			name: "nxdomain response",
			header: Header{
				ID:    0x5678,
				QR:    true,
				RCode: RCodeNameError,
			},
		},
		{
			name: "truncated response",
			header: Header{
				ID:    0xABCD,
				QR:    true,
				TC:    true,
				RA:    true,
				RCode: RCodeNoError,
			},
		},
		{
			name: "status opcode",
			header: Header{
				ID:     0x0001,
				QR:     false,
				Opcode: OpcodeStatus,
				RD:     false,
			},
		},
		{
			name: "server failure",
			header: Header{
				ID:    0x0002,
				QR:    true,
				RCode: RCodeServerFailure,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.header.Encode()

			if len(encoded) != 12 {
				t.Fatalf("expected 12 bytes, got %d", len(encoded))
			}

			decoded, err := DecodeHeader(encoded)
			if err != nil {
				t.Fatalf("DecodeHeader error: %v", err)
			}

			if decoded != tt.header {
				t.Errorf("round-trip mismatch\n  got:  %+v\n  want: %+v", decoded, tt.header)
			}
		})
	}
}

func TestDecodeHeaderTooShort(t *testing.T) {
	_, err := DecodeHeader([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for short input, got nil")
	}
}

// ── Question ──────────────────────────────────────────────────────────────────

func TestDecodeQuestion(t *testing.T) {
	tests := []struct {
		name         string
		msg          []byte
		offset       int
		wantQuestion Question
		wantOffset   int
		wantErr      bool
	}{
		{
			name: "simple A query",
			msg: append(
				[]byte{6, 'g', 'o', 'o', 'g', 'l', 'e', 3, 'c', 'o', 'm', 0},
				[]byte{0x00, 0x01, 0x00, 0x01}..., // type A, class IN
			),
			offset:       0,
			wantQuestion: Question{Name: "google.com.", Type: 1, Class: 1},
			wantOffset:   16,
		},
		{
			name: "question not at offset zero",
			msg: append(
				[]byte{0xFF, 0xFF, 0xFF},
				append(
					[]byte{6, 'g', 'o', 'o', 'g', 'l', 'e', 3, 'c', 'o', 'm', 0},
					[]byte{0x00, 0x1C, 0x00, 0x01}..., // type AAAA, class IN
				)...,
			),
			offset:       3,
			wantQuestion: Question{Name: "google.com.", Type: 28, Class: 1},
			wantOffset:   19,
		},
		{
			name: "question with pointer compression in name",
			msg: append(
				[]byte{6, 'g', 'o', 'o', 'g', 'l', 'e', 3, 'c', 'o', 'm', 0},
				append(
					[]byte{0xC0, 0x00},                // pointer to offset 0
					[]byte{0x00, 0x01, 0x00, 0x01}..., // type A, class IN
				)...,
			),
			offset:       12,
			wantQuestion: Question{Name: "google.com.", Type: 1, Class: 1},
			wantOffset:   18,
		},
		{
			name:    "truncated — missing type and class",
			msg:     []byte{6, 'g', 'o', 'o', 'g', 'l', 'e', 3, 'c', 'o', 'm', 0},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "truncated — only 2 bytes after name",
			msg:     append([]byte{3, 'c', 'o', 'm', 0}, []byte{0x00, 0x01}...),
			offset:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, offset, err := DecodeQuestion(tt.msg, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeQuestion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.wantQuestion {
				t.Errorf("question: got %+v, want %+v", got, tt.wantQuestion)
			}
			if offset != tt.wantOffset {
				t.Errorf("offset: got %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

func TestEncodeQuestionRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		q    Question
	}{
		{name: "A query", q: Question{Name: "google.com.", Type: 1, Class: 1}},
		{name: "AAAA query", q: Question{Name: "example.com.", Type: 28, Class: 1}},
		{name: "MX query", q: Question{Name: "mail.example.com.", Type: 15, Class: 1}},
		{name: "root query", q: Question{Name: ".", Type: 1, Class: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeQuestion(tt.q)
			if err != nil {
				t.Fatalf("EncodeQuestion() error: %v", err)
			}
			got, _, err := DecodeQuestion(encoded, 0)
			if err != nil {
				t.Fatalf("DecodeQuestion() error: %v", err)
			}
			if got != tt.q {
				t.Errorf("round-trip: got %+v, want %+v", got, tt.q)
			}
		})
	}
}

// ── RR ────────────────────────────────────────────────────────────────────────

func TestRRRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		rr   RR
	}{
		{
			name: "A record",
			rr: RR{
				Header: RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 300},
				Data:   ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
			},
		},
		{
			name: "AAAA record",
			rr: RR{
				Header: RRHeader{Name: "google.com.", Type: 28, Class: 1, TTL: 300},
				Data:   AAAARecord{IP: net.ParseIP("2a00:1450:4005:802::200e")},
			},
		},
		{
			name: "CNAME record",
			rr: RR{
				Header: RRHeader{Name: "www.example.com.", Type: 5, Class: 1, TTL: 3600},
				Data:   CNAMERecord{Name: "example.com."},
			},
		},
		{
			name: "NS record",
			rr: RR{
				Header: RRHeader{Name: "example.com.", Type: 2, Class: 1, TTL: 86400},
				Data:   NSRecord{Name: "ns1.example.com."},
			},
		},
		{
			name: "MX record",
			rr: RR{
				Header: RRHeader{Name: "example.com.", Type: 15, Class: 1, TTL: 3600},
				Data:   MXRecord{Preference: 10, Exchange: "mail.example.com."},
			},
		},
		{
			name: "SOA record",
			rr: RR{
				Header: RRHeader{Name: "example.com.", Type: 6, Class: 1, TTL: 3600},
				Data: SOARecord{
					MName:   "ns1.example.com.",
					RName:   "admin.example.com.",
					Serial:  2024010101,
					Refresh: 3600,
					Retry:   900,
					Expire:  604800,
					Minimum: 300,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.rr.Encode()
			if err != nil {
				t.Fatalf("RR.Encode() error: %v", err)
			}
			got, _, err := DecodeRR(encoded, 0)
			if err != nil {
				t.Fatalf("DecodeRR() error: %v", err)
			}
			// re-encode the decoded RR and compare bytes
			reEncoded, err := got.Encode()
			if err != nil {
				t.Fatalf("re-encode error: %v", err)
			}
			if !bytesEqual(encoded, reEncoded) {
				t.Errorf("round-trip byte mismatch\n  got:  %v\n  want: %v", reEncoded, encoded)
			}
		})
	}
}

// ── Message ───────────────────────────────────────────────────────────────────

func TestMessageRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "query with one question",
			msg: Message{
				Header: Header{
					ID:      0x1234,
					QR:      false,
					Opcode:  OpcodeQuery,
					RD:      true,
					QDCount: 1,
				},
				Questions: []Question{
					{Name: "google.com.", Type: 1, Class: 1},
				},
			},
		},
		{
			name: "response with A record answer",
			msg: Message{
				Header: Header{
					ID:      0x1234,
					QR:      true,
					Opcode:  OpcodeQuery,
					AA:      false,
					RD:      true,
					RA:      true,
					RCode:   RCodeNoError,
					QDCount: 1,
					ANCount: 1,
				},
				Questions: []Question{
					{Name: "google.com.", Type: 1, Class: 1},
				},
				Answers: []RR{
					{
						Header: RRHeader{Name: "google.com.", Type: 1, Class: 1, TTL: 300},
						Data:   ARecord{IP: net.IPv4(142, 250, 74, 46).To4()},
					},
				},
			},
		},
		{
			name: "response with multiple sections",
			msg: Message{
				Header: Header{
					ID: 0x5678, QR: true, RD: true, RA: true,
					QDCount: 1, ANCount: 1, NSCount: 1, ARCount: 1,
				},
				Questions: []Question{
					{Name: "example.com.", Type: 1, Class: 1},
				},
				Answers: []RR{
					{
						Header: RRHeader{Name: "example.com.", Type: 1, Class: 1, TTL: 300},
						Data:   ARecord{IP: net.IPv4(93, 184, 216, 34).To4()},
					},
				},
				Authorities: []RR{
					{
						Header: RRHeader{Name: "example.com.", Type: 2, Class: 1, TTL: 86400},
						Data:   NSRecord{Name: "ns1.example.com."},
					},
				},
				Additionals: []RR{
					{
						Header: RRHeader{Name: "ns1.example.com.", Type: 1, Class: 1, TTL: 86400},
						Data:   ARecord{IP: net.IPv4(205, 251, 196, 1).To4()},
					},
				},
			},
		},
		{
			name: "NXDOMAIN response",
			msg: Message{
				Header: Header{
					ID: 0xABCD, QR: true, RCode: RCodeNameError, QDCount: 1,
				},
				Questions: []Question{
					{Name: "doesnotexist.example.com.", Type: 1, Class: 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.msg.Encode()
			if err != nil {
				t.Fatalf("Message.Encode() error: %v", err)
			}
			got, err := DecodeMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeMessage() error: %v", err)
			}
			reEncoded, err := got.Encode()
			if err != nil {
				t.Fatalf("re-encode error: %v", err)
			}
			if !bytesEqual(encoded, reEncoded) {
				t.Errorf("round-trip byte mismatch\n  original:  %v\n  reEncoded: %v", encoded, reEncoded)
			}
		})
	}
}

func TestDecodeMessageTooShort(t *testing.T) {
	_, err := DecodeMessage([]byte{0x00, 0x01, 0x02})
	if err == nil {
		t.Error("expected error for short input, got nil")
	}
}
