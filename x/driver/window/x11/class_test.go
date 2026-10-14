package x11

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWMClassBytes(t *testing.T) {
	raw := wmClassBytes("Demo", "br.tec.lew.demo")
	want := append(append([]byte("Demo"), 0), append([]byte("br.tec.lew.demo"), 0)...)
	if string(raw) != string(want) {
		require.FailNowf(t, "", "class %q", raw)
	}
	if wmClassBytes("", "") != nil {
		require.FailNow(t, fmt.Sprint("empty class"))
	}
}
