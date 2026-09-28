package jni

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModifiedUTF8RoundTrip(t *testing.T) {
	samples := []string{
		"",
		"hello",
		"café",
		"a\x00b",
		"😀",
		"a\x00😀b",
	}
	for _, s := range samples {
		got := decodeMUTF8(encodeMUTF8(nil, s))
		require.Equal(t, s, got, s)
	}
}

func TestModifiedUTF8EmojiBytes(t *testing.T) {
	// U+1F600 is the surrogate pair U+D83D U+DE00 in modified UTF-8.
	raw := []byte{0xED, 0xA0, 0xBD, 0xED, 0xB8, 0x80}
	require.Equal(t, "😀", decodeMUTF8(raw))
	require.Equal(t, raw, encodeMUTF8(nil, "😀"))
}
