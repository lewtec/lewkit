package taskgroup

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLineWriterLazyUntilVisible(t *testing.T) {
	hub := newLiveHub()
	w := newLineWriter(hub, func(string) {})
	_, err := w.Write([]byte("\x1b[2K\r"))
	require.NoError(t, err)
	assert.Empty(t, hub.snapshot())
	_, err = w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, []string{"hello"}, hub.snapshot())
}

func TestLineWriterCommitOnNewline(t *testing.T) {
	hub := newLiveHub()
	var committed []string
	w := newLineWriter(hub, func(s string) { committed = append(committed, s) })
	_, err := w.Write([]byte("10%\r100%\ndone\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"100%", "done"}, committed)
	assert.Empty(t, hub.snapshot())
}

func TestLineWriterCloseCommitsLeftover(t *testing.T) {
	hub := newLiveHub()
	var committed []string
	w := newLineWriter(hub, func(s string) { committed = append(committed, s) })
	_, err := w.Write([]byte("almost"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.Equal(t, []string{"almost"}, committed)
	assert.Empty(t, hub.snapshot())
	require.NoError(t, w.Close())
}

func TestLineWriterReadFromCloses(t *testing.T) {
	hub := newLiveHub()
	var committed []string
	w := newLineWriter(hub, func(s string) { committed = append(committed, s) })
	n, err := w.ReadFrom(strings.NewReader("10%\r100%"))
	require.NoError(t, err)
	assert.Equal(t, int64(8), n)
	assert.Equal(t, []string{"100%"}, committed)
}

func TestLineWriterTwoIndependentRows(t *testing.T) {
	hub := newLiveHub()
	a := newLineWriter(hub, func(string) {})
	b := newLineWriter(hub, func(string) {})
	_, err := a.Write([]byte("alpha"))
	require.NoError(t, err)
	_, err = b.Write([]byte("beta"))
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"}, hub.snapshot())
	_, err = a.Write([]byte("\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"beta"}, hub.snapshot())
}

func TestLiveHubAbandonAll(t *testing.T) {
	hub := newLiveHub()
	var printed []string
	w := newLineWriter(hub, func(s string) { printed = append(printed, s) })
	_, err := w.Write([]byte("leftover"))
	require.NoError(t, err)
	assert.Equal(t, []string{"leftover"}, hub.abandonAll())
	assert.Empty(t, printed)
	_, err = w.Write([]byte("x"))
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestLineWriterFromNoSession(t *testing.T) {
	w := LineWriterFrom(t.Context())
	lw, ok := w.(*lineWriter)
	require.True(t, ok)
	assert.False(t, lw.commitOnClose)
	require.NoError(t, w.Close())
}

func TestFinishedLineWriterOnlyEmitsCompleteLines(t *testing.T) {
	var buf strings.Builder
	w := newFinishedLineWriter(&buf)
	_, err := w.Write([]byte("10%\r50%\r100%\ndone\npartial"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.Equal(t, "100%\ndone\n", buf.String())
}
