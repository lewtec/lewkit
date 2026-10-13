package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotepadTypes(t *testing.T) {
	notepad, err := NewNotepad()
	require.NoError(t, err)
	next, cmd := notepad.Update(window.Key{Rune: 'x', Pressed: true})
	require.Same(t, notepad, next)
	require.Nil(t, cmd)
	assert.Contains(t, string(notepad.body), "x")
}
