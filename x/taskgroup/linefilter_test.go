package taskgroup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func feed(t *testing.T, chunks ...string) (committed []string, current string) {
	t.Helper()
	var f lineFilter
	for _, c := range chunks {
		committed = append(committed, f.write([]byte(c))...)
	}
	return committed, f.current()
}

func TestLineFilterPlain(t *testing.T) {
	got, cur := feed(t, "hello\nworld")
	assert.Equal(t, []string{"hello"}, got)
	assert.Equal(t, "world", cur)
}

func TestLineFilterCRProgress(t *testing.T) {
	got, cur := feed(t, "downloading 10%\rdownloading 50%\rdownloading 100%\ndone\n")
	assert.Equal(t, []string{"downloading 100%", "done"}, got)
	assert.Empty(t, cur)
}

func TestLineFilterCRShorterReplacesLine(t *testing.T) {
	got, cur := feed(t, "alpha 24/24\ralpha done\n", "beta 16/16\rbeta done\n")
	assert.Equal(t, []string{"alpha done", "beta done"}, got)
	assert.Empty(t, cur)
}

func TestLineFilterCRLFCommitsPrior(t *testing.T) {
	got, cur := feed(t, "hello\r\nworld\r\n")
	assert.Equal(t, []string{"hello", "world"}, got)
	assert.Empty(t, cur)
}

func TestLineFilterCRReplaceCurrent(t *testing.T) {
	_, cur := feed(t, "hello\rX")
	assert.Equal(t, "X", cur)
}

func TestLineFilterCRAcrossChunks(t *testing.T) {
	got, cur := feed(t, "10%\r", "50%\r", "100%\n")
	assert.Equal(t, []string{"100%"}, got)
	assert.Empty(t, cur)
}

func TestLineFilterStripSGR(t *testing.T) {
	got, cur := feed(t, "\x1b[31merror\x1b[0m: boom\n")
	assert.Equal(t, []string{"error: boom"}, got)
	assert.Empty(t, cur)
}
