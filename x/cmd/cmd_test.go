package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type BasicArgs struct {
	name    StringArg    `long:"name" short:"n"`
	idade   IntArg[uint] `long:"idade" short:"i"`
	verbose Count        `short:"v" long:"verbose"` // -vvv or --verbose 3
	rest    []StringArg  // that means a positional argument
}

func (a *BasicArgs) Run(ctx context.Context) error {
	fmt.Printf("name is %s, age is %d", a.name.Value(), a.idade.Value())
	for _, arg := range a.rest {
		fmt.Printf("- %s", arg.Value())
	}
	return nil
}

func TestParseUtil(t *testing.T) {
	args, err := Parse[BasicArgs]("--name", "Lucas", "--idade", "26", "-vvv", "leftover")
	require.NoError(t, err)
	assert.Equal(t, args.name.Value(), "Lucas")
	assert.Equal(t, args.idade.Value(), uint(26))
	assert.Equal(t, args.verbose.Value(), 3)
	assert.Equal(t, Values(args.rest), []string{"leftover"})
}

func TestArgsBasic(t *testing.T) {
	var cmd Command[BasicArgs]

	assert.NoError(t, cmd.Parse("--name", "Lucas", "--idade", "26", "-vvv", "leftover", "foo", "bar"))
	assert.Equal(t, cmd.args.name.Value(), "Lucas")
	assert.Equal(t, cmd.args.idade.Value(), uint(26))
	assert.Equal(t, cmd.args.verbose.Value(), 3)
	assert.Equal(t, Values(cmd.args.rest), []string{"leftover", "foo", "bar"})
}

func TestArgsForms(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want func(*testing.T, BasicArgs)
	}{
		{
			name: "short and equals",
			args: []string{"-n", "Lucas", "--idade=26", "-v", "-v", "x"},
			want: func(t *testing.T, a BasicArgs) {
				assert.Equal(t, a.name.Value(), "Lucas")
				assert.Equal(t, a.idade.Value(), uint(26))
				assert.Equal(t, a.verbose.Value(), 2)
				assert.Equal(t, Values(a.rest), []string{"x"})
			},
		},
		{
			name: "clustered short value",
			args: []string{"-nLucas", "-i26"},
			want: func(t *testing.T, a BasicArgs) {
				assert.Equal(t, a.name.Value(), "Lucas")
				assert.Equal(t, a.idade.Value(), uint(26))
			},
		},
		{
			name: "count long value",
			args: []string{"--verbose", "3"},
			want: func(t *testing.T, a BasicArgs) {
				assert.Equal(t, a.verbose.Value(), 3)
			},
		},
		{
			name: "count equals",
			args: []string{"--verbose=2"},
			want: func(t *testing.T, a BasicArgs) {
				assert.Equal(t, a.verbose.Value(), 2)
			},
		},
		{
			name: "dash dash",
			args: []string{"--name", "a", "--", "-still", "pos"},
			want: func(t *testing.T, a BasicArgs) {
				assert.Equal(t, a.name.Value(), "a")
				assert.Equal(t, Values(a.rest), []string{"-still", "pos"})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cmd Command[BasicArgs]
			require.NoError(t, cmd.Parse(tc.args...))
			tc.want(t, cmd.Args())
		})
	}
}

func TestArgsErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want error
	}{
		{name: "unknown long", args: []string{"--nope"}, want: ErrUnknownFlag},
		{name: "unknown short", args: []string{"-z"}, want: ErrUnknownFlag},
		{name: "missing value", args: []string{"--name"}, want: ErrMissingValue},
		{name: "bad int", args: []string{"--idade", "x"}, want: ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cmd Command[BasicArgs]
			err := cmd.Parse(tc.args...)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

type flagArgs struct {
	force Flag `long:"force" short:"f"`
	rest  []StringArg
}

func TestFlagOnce(t *testing.T) {
	var cmd Command[flagArgs]
	require.NoError(t, cmd.Parse("--force", "keep"))
	assert.True(t, cmd.args.force.Value())
	assert.Equal(t, Values(cmd.args.rest), []string{"keep"})
}

func TestFlagTwice(t *testing.T) {
	var cmd Command[flagArgs]
	err := cmd.Parse("-f", "-f")
	assert.ErrorIs(t, err, ErrInvalidArgument)
}

type posArgs struct {
	first StringArg
	rest  []StringArg
}

func TestPositionals(t *testing.T) {
	var cmd Command[posArgs]
	require.NoError(t, cmd.Parse("a", "b", "c"))
	assert.Equal(t, cmd.args.first.Value(), "a")
	assert.Equal(t, Values(cmd.args.rest), []string{"b", "c"})
}

type tagsArgs struct {
	tag []StringArg `long:"tag" short:"t"`
}

func TestRepeatable(t *testing.T) {
	var cmd Command[tagsArgs]
	require.NoError(t, cmd.Parse("--tag", "a", "-t", "b", "--tag=c"))
	assert.Equal(t, Values(cmd.args.tag), []string{"a", "b", "c"})
}
