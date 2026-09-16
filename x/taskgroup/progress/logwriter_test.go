package progress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinePrinterSplitsOnNewline(t *testing.T) {
	var got []string
	w := &linePrinter{print: func(s string) { got = append(got, s) }}
	_, err := w.Write([]byte("hello\nwor"))
	require.NoError(t, err)
	_, err = w.Write([]byte("ld\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, got)
}

func TestLinePrinterHoldsPartial(t *testing.T) {
	var got []string
	w := &linePrinter{print: func(s string) { got = append(got, s) }}
	_, err := w.Write([]byte("no-nl"))
	require.NoError(t, err)
	assert.Empty(t, got)
	w.close()
	_, err = w.Write([]byte("after\n"))
	require.NoError(t, err)
	assert.Empty(t, got)
}
