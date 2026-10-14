//go:build linux && !android

package gtk

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGTK4StaysOutOfWebKit41Process(t *testing.T) {
	if !blockGTK4Choice(false, false, true) {
		require.FailNow(t, fmt.Sprint("webkit 4.1 alone should keep GTK 4 out"))
	}
	if blockGTK4Choice(false, true, true) {
		require.FailNow(t, fmt.Sprint("webkit 6 should allow GTK 4"))
	}
	if blockGTK4Choice(false, false, false) {
		require.FailNow(t, fmt.Sprint("no webkit should allow GTK 4 dialogs"))
	}
	if !blockGTK4Choice(true, true, false) {
		require.FailNow(t, fmt.Sprint("a loaded GTK 3 blocks GTK 4"))
	}
}

func TestPollBeforeInit(t *testing.T) {
	if Poll() != 0 {
		require.FailNow(t, fmt.Sprint("poll before Ensure dispatched work"))
	}
}

func TestAvailableOrSkip(t *testing.T) {
	if err := Available(t.Context()); err != nil {
		t.Skip(err)
	}
}
