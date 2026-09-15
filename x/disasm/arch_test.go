package disasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArchitecture(t *testing.T) {
	cases := []struct {
		in   string
		want Architecture
	}{
		{in: "x86", want: ArchitectureX86},
		{in: "AMD64", want: ArchitectureX86},
		{in: "arm64", want: ArchitectureAArch64},
		{in: "aarch64", want: ArchitectureAArch64},
		{in: "riscv", want: ArchitectureRISCV},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseArchitecture(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseArchitectureUnknown(t *testing.T) {
	_, err := ParseArchitecture("itanium")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown architecture")
}

func TestParseMode(t *testing.T) {
	got, err := ParseMode("64")
	require.NoError(t, err)
	assert.Equal(t, Mode64, got)

	got, err = ParseMode("thumb,v8")
	require.NoError(t, err)
	assert.Equal(t, ModeThumb|ModeV8, got)

	got, err = ParseMode("")
	require.NoError(t, err)
	assert.Equal(t, Mode(0), got)
}

func TestParseModeUnknown(t *testing.T) {
	_, err := ParseMode("99")
	require.Error(t, err)
}

func TestParseSyntax(t *testing.T) {
	got, err := ParseSyntax("intel")
	require.NoError(t, err)
	assert.Equal(t, SyntaxIntel, got)

	got, err = ParseSyntax("at&t")
	require.NoError(t, err)
	assert.Equal(t, SyntaxATT, got)
}
