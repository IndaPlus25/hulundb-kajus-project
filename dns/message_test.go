package dns

import "testing"

func FuzzDecodeName(f *testing.F) {
	// seed corpus — valid inputs
	f.Add([]byte{3, 'w', 'w', 'w', 0}, 0)
	f.Add([]byte{0}, 0)

	f.Fuzz(func(t *testing.T, msg []byte, offset int) {
		if offset < 0 || offset >= len(msg) {
			return
		}
		name, n, err := DecodeName(msg, offset)
		if err != nil {
			return // error path must not panic — that's the invariant
		}
		// if it succeeded, the result must be sane
		if n <= 0 || n > len(msg)-offset {
			t.Errorf("consumed %d bytes, but only %d available", n, len(msg)-offset)
		}
		_ = name
	})
}
