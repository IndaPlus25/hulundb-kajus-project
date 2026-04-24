package dns

import (
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
