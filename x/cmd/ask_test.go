package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsksUsesDefaultsAndEnumChoices(t *testing.T) {
	type args struct {
		name  StringArg      `long:"name" help:"publisher name" default:"Anon"`
		color EnumArg[color] `long:"color" help:"tint" default:"green"`
		force Flag           `long:"force"`
	}
	got := ParseOK[args](t)
	asks, err := Asks(&got)
	require.NoError(t, err)
	require.Len(t, asks, 2)
	assert.Equal(t, "name", asks[0].Name)
	assert.Equal(t, "publisher name", asks[0].Prompt)
	assert.Equal(t, "Anon", asks[0].Default)
	assert.Empty(t, asks[0].Choices)
	assert.Equal(t, "color", asks[1].Name)
	assert.Equal(t, "tint", asks[1].Prompt)
	assert.Equal(t, "green", asks[1].Default)
	assert.Equal(t, []string{"red", "green", "blue"}, asks[1].Choices)

	require.NoError(t, asks[1].Apply("blue"))
	assert.Equal(t, colorBlue, got.color.Value())
	assert.True(t, got.color.ArgSet())
	left, err := Asks(&got)
	require.NoError(t, err)
	require.Len(t, left, 1)
	assert.Equal(t, "name", left[0].Name)
}

func TestAsksSkipsExplicitValues(t *testing.T) {
	type args struct {
		name  StringArg      `long:"name" help:"publisher name" default:"Anon"`
		color EnumArg[color] `long:"color" help:"tint" default:"green"`
	}
	got := ParseOK[args](t, "--name", "Acme", "--color", "red")
	asks, err := Asks(&got)
	require.NoError(t, err)
	assert.Empty(t, asks)
}

func TestAsksKeepsBlankExplicitText(t *testing.T) {
	type args struct {
		name StringArg `long:"name" help:"publisher name" default:"Anon"`
	}
	got := ParseOK[args](t, "--name", "")
	asks, err := Asks(&got)
	require.NoError(t, err)
	require.Len(t, asks, 1)
	assert.Equal(t, "publisher name", asks[0].Prompt)
	assert.Equal(t, "Anon", asks[0].Default)
}
