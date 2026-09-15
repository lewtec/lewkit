package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type color string

const (
	colorRed   color = "red"
	colorGreen color = "green"
	colorBlue  color = "blue"
)

func (color) Values() []color {
	return []color{colorRed, colorGreen, colorBlue}
}

func TestEnumArg(t *testing.T) {
	type args struct {
		color EnumArg[color] `long:"color"`
	}
	cases := []struct {
		name string
		in   string
		want color
		err  error
	}{
		{name: "red", in: "red", want: colorRed},
		{name: "green", in: "green", want: colorGreen},
		{name: "blue", in: "blue", want: colorBlue},
		{name: "empty", in: "", err: ErrInvalidArgument},
		{name: "unknown", in: "purple", err: ErrInvalidArgument},
		{name: "wrong case", in: "Red", err: ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse[args]("--color", tc.in)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.color.Value())
		})
	}
}

func TestEnumDefault(t *testing.T) {
	type args struct {
		color EnumArg[color] `long:"color" default:"green"`
	}
	got := ParseOK[args](t)
	assert.Equal(t, colorGreen, got.color.Value())
}

func TestEnumPositional(t *testing.T) {
	type args struct {
		color EnumArg[color]
	}
	got := ParseOK[args](t, "blue")
	assert.Equal(t, colorBlue, got.color.Value())
}

func TestUsageChoices(t *testing.T) {
	type args struct {
		color EnumArg[color] `long:"color" help:"tint"`
	}
	text, err := Usage[args]("tool")
	require.NoError(t, err)
	assert.Contains(t, text, "tint (choices: red, green, blue)")
}

var (
	_ Parser         = (*EnumArg[color])(nil)
	_ Arg[color]     = (*EnumArg[color])(nil)
	_ ArgChooser     = EnumArg[color]{}
)
