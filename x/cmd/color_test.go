package cmd

import (
	"testing"

	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorArg(t *testing.T) {
	type args struct {
		accent ColorArg `long:"accent" default:""`
	}
	got := ParseOK[args](t)
	assert.False(t, got.accent.IsSet())
	got, err := Parse[args]("--accent", "#0d3559")
	require.NoError(t, err)
	assert.True(t, got.accent.IsSet())
	assert.Equal(t, lewimage.Color{13, 53, 89, 255}, got.accent.Value())
	_, err = Parse[args]("--accent", "nope")
	assert.ErrorIs(t, err, ErrInvalidArgument)
}
