package progress

import (
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTree(t *testing.T) {
	got := Format([]taskgroup.Node{
		{ID: 1, Name: "window.Driver", Message: "=> window_cocoa", State: taskgroup.Done, Pool: taskgroup.Control, LiveChildren: 2},
		{ID: 2, Parent: 1, Name: "window_cocoa", Message: "selected", State: taskgroup.Done, Pool: taskgroup.CPU},
		{ID: 3, Parent: 1, Name: "window_mem", Message: "available", State: taskgroup.Done, Pool: taskgroup.CPU},
	}, 80)
	require.NotEmpty(t, got)
	assert.Contains(t, got, "window.Driver")
	assert.Contains(t, got, "=> window_cocoa")
	assert.Contains(t, got, "├ ")
	assert.Contains(t, got, "└ ")
	assert.Contains(t, got, "window_cocoa")
	assert.Contains(t, got, "window_mem")
}
