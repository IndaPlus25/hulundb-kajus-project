package dns

import (
	"fmt"
	"strings"
)

const (
	maxNameLength  = 255 // RFC 1035 §2.3.4
	maxLabelLength = 63
	maxPtrDepth    = 128  // your loop-detection budget
	ptrMask        = 0xC0 // top two bits set → pointer
	ptrOffsetMask  = 0x3F // bottom 6 bits of first pointer byte
)

// DecodeName parses a DNS name from msg starting at offset.
// Returns the name (e.g. "www.example.com."), the number of
// bytes consumed at `offset` (NOT total bytes followed), and any error.
func DecodeName(msg []byte, offset int) (string, int, error) {
	cur := offset
	consumed := -1
	ptrDepth := 0
	labels := []string{}

	for {
		if cur < 0 || cur >= len(msg) {
			return "", 0, fmt.Errorf("offset out of bounds")
		}

		b := msg[cur]
		if b == 0x00 { //reached end of the name
			if consumed == -1 {
				consumed = cur - offset + 1
			}

			break
		} else if b&0xC0 == 0xC0 { // check all error codes, might be wrong
			if cur+1 >= len(msg) {
				return "", 0, fmt.Errorf("truncated compression pointer")
			}
			if consumed == -1 {
				consumed = cur - offset + 2
			}
			ptrDepth++
			if ptrDepth > maxPtrDepth {
				return "", 0, fmt.Errorf("pointer depth error")
			}

			target := int(b&0x3F)<<8 | int(msg[cur+1])

			if target < 0 || target >= len(msg) {
				return "", 0, fmt.Errorf("pointer target out of bounds")
			}
			cur = target

		} else if b&0xC0 == 0x00 {
			labelLen := int(b)
			if cur+1+labelLen > len(msg) {
				return "", 0, fmt.Errorf("truncated compression pointer")
			}
			if labelLen > maxLabelLength {
				return "", 0, fmt.Errorf("label too long")
			}

			labels = append(labels, string(msg[cur+1:cur+1+labelLen]))
			cur += 1 + labelLen

		} else {
			return "", 0, fmt.Errorf("reserved label typ")
		}

	}
	name := strings.Join(labels, ".") + "."
	return name, consumed, nil
}

// EncodeName encodes a DNS name string to wire format.
// Input may have a trailing dot or not: both "example.com" and
// "example.com." are valid.
func EncodeName(name string) ([]byte, error) {

	labels := strings.Split(name, ".")

	var buf []byte

	if name == "" {
		return nil, fmt.Errorf("empty name: use \".\" for root")
	}
	if name == "." { //Root edge case
		buf = append(buf, 0x00)
		return buf, nil
	}

	for i, label := range labels {
		if len(label) > maxLabelLength {
			return nil, fmt.Errorf("label too long: %q", label)
		}

		if label == "" {
			if i == len(labels)-1 {
				continue
			} else {
				return nil, fmt.Errorf("invalid position for empty label")
			}
		}

		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)

	}
	buf = append(buf, 0x00)

	if len(buf) > maxNameLength {
		return nil, fmt.Errorf("name too long:")
	}

	return buf, nil
}
