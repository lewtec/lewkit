package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type BasicArgs struct {
	name    StringArg    `long:"name" short:"n" default:""`
	idade   IntArg[uint] `long:"idade" short:"i" default:"0"`
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

type globals struct {
	verbose Count `short:"v" long:"verbose"`
}

type addCmd struct {
	globals
	name StringArg `long:"name"`
	rest []StringArg
}

type rmCmd struct {
	path StringArg
}

type appArgs struct {
	globals
	add *addCmd
	rm  *rmCmd
}

func TestSubcommandParentFlag(t *testing.T) {
	args, err := Parse[appArgs]("-v", "add", "--name", "x", "file")
	require.NoError(t, err)
	assert.Equal(t, args.verbose.Value(), 1)
	require.NotNil(t, args.add)
	assert.Nil(t, args.rm)
	assert.Equal(t, args.add.name.Value(), "x")
	assert.Equal(t, Values(args.add.rest), []string{"file"})
	assert.Equal(t, args.add.verbose.Value(), 0)
}

func TestSubcommandChildFlag(t *testing.T) {
	args, err := Parse[appArgs]("add", "-v", "--name", "x")
	require.NoError(t, err)
	assert.Equal(t, args.verbose.Value(), 0)
	require.NotNil(t, args.add)
	assert.Equal(t, args.add.verbose.Value(), 1)
	assert.Equal(t, args.add.name.Value(), "x")
}

func TestSubcommandRm(t *testing.T) {
	args, err := Parse[appArgs]("rm", "gone")
	require.NoError(t, err)
	require.NotNil(t, args.rm)
	assert.Nil(t, args.add)
	assert.Equal(t, args.rm.path.Value(), "gone")
}

func TestSubcommandOnlyParent(t *testing.T) {
	args, err := Parse[appArgs]("-vv")
	require.NoError(t, err)
	assert.Equal(t, args.verbose.Value(), 2)
	assert.Nil(t, args.add)
	assert.Nil(t, args.rm)
}

func TestSubcommandUnknown(t *testing.T) {
	_, err := Parse[appArgs]("nope")
	assert.ErrorIs(t, err, ErrUnknownCommand)
}

type inheritChild struct {
	name StringArg `long:"name" default:""`
	rest []StringArg
}

type inheritApp struct {
	verbose Count     `short:"v" long:"verbose"`
	force   Flag      `long:"force" short:"f"`
	name    StringArg `long:"label" default:""`
	add     *inheritChild
	mid     *inheritMid
}

type inheritMid struct {
	inner *inheritChild
}

func TestInheritParentFlagAfterCommand(t *testing.T) {
	args, err := Parse[inheritApp]("add", "-v", "--name", "x", "file")
	require.NoError(t, err)
	assert.Equal(t, 1, args.verbose.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, "x", args.add.name.Value())
	assert.Equal(t, []string{"file"}, Values(args.add.rest))
}

func TestInheritParentFlagBeforeAndAfter(t *testing.T) {
	args, err := Parse[inheritApp]("-v", "add", "-v", "--name", "x")
	require.NoError(t, err)
	assert.Equal(t, 2, args.verbose.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, "x", args.add.name.Value())
}

func TestInheritParentValueAfterCommand(t *testing.T) {
	args, err := Parse[inheritApp]("add", "--label", "root", "--name", "x")
	require.NoError(t, err)
	assert.Equal(t, "root", args.name.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, "x", args.add.name.Value())
}

func TestInheritStopsAfterDashDash(t *testing.T) {
	args, err := Parse[inheritApp]("add", "--name", "x", "--", "-v", "--force")
	require.NoError(t, err)
	assert.Equal(t, 0, args.verbose.Value())
	assert.False(t, args.force.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, []string{"-v", "--force"}, Values(args.add.rest))
}

func TestInheritStopsAfterParentDashDash(t *testing.T) {
	args, err := Parse[inheritApp]("--", "add", "-v")
	require.NoError(t, err)
	assert.Equal(t, 0, args.verbose.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, []string{"-v"}, Values(args.add.rest))
}

func TestInheritNestedCommand(t *testing.T) {
	args, err := Parse[inheritApp]("mid", "inner", "-vv", "--name", "z")
	require.NoError(t, err)
	assert.Equal(t, 2, args.verbose.Value())
	require.NotNil(t, args.mid)
	require.NotNil(t, args.mid.inner)
	assert.Equal(t, "z", args.mid.inner.name.Value())
}

func TestInheritUnknownStillErrors(t *testing.T) {
	_, err := Parse[inheritApp]("add", "--nope")
	assert.ErrorIs(t, err, ErrUnknownFlag)
}

func TestRequiredParentFlagAfterCommand(t *testing.T) {
	type child struct {
		rest []StringArg
	}
	type parent struct {
		name StringArg `long:"name"`
		add  *child
	}
	args, err := Parse[parent]("add", "--name", "x", "file")
	require.NoError(t, err)
	assert.Equal(t, "x", args.name.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, []string{"file"}, Values(args.add.rest))
}

func TestInheritedHelpSkipsChildRequired(t *testing.T) {
	type child struct {
		path StringArg `long:"path"`
	}
	type parent struct {
		help Flag `long:"help"`
		add  *child
	}
	got, err := Parse[parent]("add", "--help")
	require.NoError(t, err)
	assert.True(t, got.help.Value())
	require.NotNil(t, got.add)
}

type plusApp struct {
	plus *addCmd `cmd:"plus"`
}

func TestSubcommandTag(t *testing.T) {
	args, err := Parse[plusApp]("plus", "--name", "n")
	require.NoError(t, err)
	require.NotNil(t, args.plus)
	assert.Equal(t, args.plus.name.Value(), "n")
}

type nestedInner struct {
	name StringArg `long:"name"`
}

type nestedMid struct {
	inner *nestedInner
}

type nestedApp struct {
	mid *nestedMid
}

func TestNestedSubcommand(t *testing.T) {
	args, err := Parse[nestedApp]("mid", "inner", "--name", "z")
	require.NoError(t, err)
	require.NotNil(t, args.mid)
	require.NotNil(t, args.mid.inner)
	assert.Equal(t, args.mid.inner.name.Value(), "z")
}

type mixedArgs struct {
	add  *addCmd
	rest []StringArg
}

func TestCommandMixPositional(t *testing.T) {
	_, err := Parse[mixedArgs]("add")
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

type flattenInner struct {
	name StringArg `long:"name"`
}

type flattenOuter struct {
	force Flag         `long:"force"`
	inner flattenInner `flatten:""`
}

func TestFlattenTag(t *testing.T) {
	args, err := Parse[flattenOuter]("--force", "--name", "x")
	require.NoError(t, err)
	assert.True(t, args.force.Value())
	assert.Equal(t, "x", args.inner.name.Value())
}

type defaultFlagArgs struct {
	name  StringArg   `long:"name" default:"Lucas"`
	port  IntArg[int] `long:"port" default:"8080"`
	force Flag        `long:"force" default:"true"`
	level Count       `long:"verbose" short:"v" default:"2"`
	path  StringArg   `default:"."`
}

func TestDefaultApplied(t *testing.T) {
	args, err := Parse[defaultFlagArgs]()
	require.NoError(t, err)
	assert.Equal(t, "Lucas", args.name.Value())
	assert.Equal(t, 8080, args.port.Value())
	assert.True(t, args.force.Value())
	assert.Equal(t, 2, args.level.Value())
	assert.Equal(t, ".", args.path.Value())
}

func TestDefaultOverridden(t *testing.T) {
	args, err := Parse[defaultFlagArgs]("--name", "Ada", "--port", "9", "--verbose", "0", "src")
	require.NoError(t, err)
	assert.Equal(t, "Ada", args.name.Value())
	assert.Equal(t, 9, args.port.Value())
	assert.True(t, args.force.Value())
	assert.Equal(t, 0, args.level.Value())
	assert.Equal(t, "src", args.path.Value())
}

type badDefaultArgs struct {
	port IntArg[int] `long:"port" default:"nope"`
}

func TestDefaultInvalid(t *testing.T) {
	_, err := Parse[badDefaultArgs]()
	assert.ErrorIs(t, err, ErrInvalidArgument)
}

type defaultOnRest struct {
	rest []StringArg `default:"x"`
}

func TestDefaultOnRest(t *testing.T) {
	_, err := Parse[defaultOnRest]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

type defaultOnCmd struct {
	add *addCmd `default:"1"`
}

func TestDefaultOnCommand(t *testing.T) {
	_, err := Parse[defaultOnCmd]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

type parentDefault struct {
	name StringArg `long:"name" default:"root"`
	add  *addCmd
}

func TestDefaultBeforeSubcommand(t *testing.T) {
	args, err := Parse[parentDefault]("add", "--name", "x", "file")
	require.NoError(t, err)
	assert.Equal(t, "root", args.name.Value())
	require.NotNil(t, args.add)
	assert.Equal(t, "x", args.add.name.Value())
	assert.Equal(t, []string{"file"}, Values(args.add.rest))
}

type requiredNameArgs struct {
	name StringArg `long:"name"`
	rest []StringArg
}

func TestRequiredFlagMissing(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "empty", args: nil},
		{name: "only positionals", args: []string{"leftover"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse[requiredNameArgs](tc.args...)
			assert.ErrorIs(t, err, ErrMissingValue)
		})
	}
}

func TestRequiredFlagPresent(t *testing.T) {
	args, err := Parse[requiredNameArgs]("--name", "Ada")
	require.NoError(t, err)
	assert.Equal(t, "Ada", args.name.Value())
}

func TestRequiredFlagsMissing(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "short value",
			run: func() error {
				_, err := Parse[struct {
					name StringArg `short:"n"`
				}]()
				return err
			},
		},
		{
			name: "int",
			run: func() error {
				_, err := Parse[struct {
					port IntArg[int] `long:"port"`
				}]()
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, tc.run(), ErrMissingValue)
		})
	}
}

func TestRequiredChildFlagMissing(t *testing.T) {
	type child struct {
		name StringArg `long:"name"`
	}
	type parent struct {
		add *child
	}
	_, err := Parse[parent]("add")
	assert.ErrorIs(t, err, ErrMissingValue)
}

func TestRequiredParentFlagMissing(t *testing.T) {
	type child struct {
		rest []StringArg
	}
	type parent struct {
		name StringArg `long:"name"`
		add  *child
	}
	_, err := Parse[parent]("add", "file")
	assert.ErrorIs(t, err, ErrMissingValue)
}

func TestHelpWithoutRequiredFlag(t *testing.T) {
	type args struct {
		help Flag      `long:"help"`
		name StringArg `long:"name"`
	}
	got, err := Parse[args]("--help")
	require.NoError(t, err)
	assert.True(t, got.help.Value())
}

func TestVersionWithoutRequiredFlag(t *testing.T) {
	type args struct {
		version Flag      `long:"version"`
		name    StringArg `long:"name"`
	}
	got, err := Parse[args]("--version")
	require.NoError(t, err)
	assert.True(t, got.version.Value())
}

type dirArg struct {
	StringArg
}

func (dirArg) ArgDefault() string { return "/tmp" }

func TestDefaultMethod(t *testing.T) {
	type args struct {
		force Flag   `long:"force"`
		n     Count  `long:"n"`
		dir   dirArg `long:"dir"`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.False(t, got.force.Value())
	assert.Equal(t, 0, got.n.Value())
	assert.Equal(t, "/tmp", got.dir.Value())
}

func TestDefaultTagOverridesMethod(t *testing.T) {
	type args struct {
		force Flag  `long:"force" default:"true"`
		n     Count `long:"n" default:"3"`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.True(t, got.force.Value())
	assert.Equal(t, 3, got.n.Value())
}
