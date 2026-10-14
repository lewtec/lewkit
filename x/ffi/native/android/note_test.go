//go:build android

package android

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotedVMSkipsLoad(t *testing.T) {
	resetNote()
	t.Cleanup(resetNote)
	if NotedVM() {
		require.FailNow(t, fmt.Sprint("vm noted before JNI_OnLoad"))
	}
	NoteVM(1)
	if !NotedVM() {
		require.FailNow(t, fmt.Sprint("vm not noted"))
	}
	NoteLoader()
	n, err := JavaVMs()
	require.NoError(t, err)
	if n != 1 {
		require.FailNowf(t, "", "vms %d", n)
	}
	on, err := OnLooper()
	require.NoError(t, err)
	if !on {
		require.FailNow(t, fmt.Sprint("loader thread is not the looper"))
	}
}
