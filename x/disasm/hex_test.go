package disasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeHex(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []byte
	}{
		{name: "plain", in: "90c3", want: []byte{0x90, 0xc3}},
		{name: "spaces", in: "90 c3", want: []byte{0x90, 0xc3}},
		{name: "0x", in: "0x90,0xc3", want: []byte{0x90, 0xc3}},
		{name: "slashx", in: `\x90\xc3`, want: []byte{0x90, 0xc3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeHex(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecodeHexErrors(t *testing.T) {
	for _, in := range []string{"", "   ", "90c"} {
		_, err := DecodeHex(in)
		assert.Error(t, err, in)
	}
}
