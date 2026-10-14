package disasm

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// DecodeHex turns a hex dump into bytes.
// Spaces, commas, 0x, and \x prefixes are ignored.
func DecodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrDecodeHexEmptyHex
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == ',':
			i++
		case i+1 < len(s) && s[i] == '\\' && (s[i+1] == 'x' || s[i+1] == 'X'):
			i += 2
		case i+1 < len(s) && s[i] == '0' && (s[i+1] == 'x' || s[i+1] == 'X'):
			i += 2
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	clean := b.String()
	if clean == "" {
		return nil, ErrDecodeHexEmptyHex
	}
	if len(clean)%2 != 0 {
		return nil, ErrOddHexLength
	}
	out, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeHexHex, err)
	}
	return out, nil
}
