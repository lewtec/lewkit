package decomp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckAcceptsEcho(t *testing.T) {
	artifact := openELF(t, []byte{0x90, 0xc3})
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	result, err := artifact.Check(t.Context(), 0x1000, echoCandidate(function, "start", "void start(void)"))
	require.NoError(t, err)
	assert.True(t, result.Matched)
	stored, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "start", stored.Name)
	assert.Contains(t, stored.Source, "void start")
}

func TestCheckRejectsMissingBlocks(t *testing.T) {
	artifact := openELF(t, []byte{0x90, 0xc3})
	wrong := `{"name":"start","signature":"void start(void)","source":"void start(void) { return; }","blocks":[]}`
	result, err := artifact.Check(t.Context(), 0x1000, wrong)
	assert.ErrorIs(t, err, ErrUnmatched)
	assert.False(t, result.Matched)
	assert.Contains(t, result.Residuals, "block 0x1000 missing")
	stored, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.NotEqual(t, "start", stored.Name)
}

func TestCheckUnmatched(t *testing.T) {
	artifact := openELF(t, []byte{0xc3})
	result, err := artifact.Check(t.Context(), 0x1000, "no json")
	assert.ErrorIs(t, err, ErrUnmatched)
	assert.False(t, result.Matched)
	assert.NotEmpty(t, result.Residuals)
}

func echoCandidate(function Function, name, signature string) string {
	source := signature + " {\n  return;\n}\n"
	blocks := ""
	for index, block := range function.Blocks {
		if index > 0 {
			blocks += ","
		}
		blocks += `{"address":"` + formatAddress(block.Address) + `","kind":"` + block.Kind + `"`
		if block.Target != 0 {
			blocks += `,"target":"` + formatAddress(block.Target) + `"`
		}
		if block.Fall != 0 {
			blocks += `,"fall":"` + formatAddress(block.Fall) + `"`
		}
		blocks += "}"
	}
	return `{"name":"` + name + `","signature":"` + signature + `","source":"` + jsonEscape(source) + `","blocks":[` + blocks + `]}`
}

func jsonEscape(text string) string {
	out := ""
	for _, char := range text {
		switch char {
		case '"':
			out += `\"`
		case '\n':
			out += `\n`
		default:
			out += string(char)
		}
	}
	return out
}
