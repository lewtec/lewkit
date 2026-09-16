package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func completeOK[T any](t *testing.T, args ...string) []string {
	t.Helper()
	suggestions, err := Complete[T](args...)
	require.NoError(t, err)
	out := make([]string, len(suggestions))
	for i, suggestion := range suggestions {
		out[i] = suggestion.Text
	}
	return out
}

func TestCompleteCommands(t *testing.T) {
	got := completeOK[appArgs](t)
	assert.ElementsMatch(t, []string{"add", "rm", "--verbose", "-v", "--"}, got)
}

func TestCompletePrefixCommand(t *testing.T) {
	assert.Equal(t, []string{"add"}, completeOK[appArgs](t, "a"))
}

func TestCompletePrefixFlag(t *testing.T) {
	assert.Equal(t, []string{"--verbose"}, completeOK[appArgs](t, "--v"))
}

func TestCompleteAfterCommand(t *testing.T) {
	got := completeOK[appArgs](t, "add", "")
	assert.ElementsMatch(t, []string{"--name", "--verbose", "-v", "--"}, got)
}

func TestCompleteParentFlagsAfterCommand(t *testing.T) {
	got := completeOK[inheritApp](t, "add", "")
	assert.ElementsMatch(t, []string{
		"--name", "--verbose", "-v", "--force", "-f", "--label", "--",
	}, got)
}

func TestCompleteAfterDashNoFlags(t *testing.T) {
	got := completeOK[inheritApp](t, "--", "")
	assert.ElementsMatch(t, []string{"add", "mid"}, got)
}

func TestCompleteUnknownStops(t *testing.T) {
	assert.Empty(t, completeOK[appArgs](t, "nope", ""))
}

func TestCompleteInvalidSpec(t *testing.T) {
	_, err := Complete[mixedArgs]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestCompleteEnumPositional(t *testing.T) {
	type args struct {
		color EnumArg[color]
	}
	assert.ElementsMatch(t, []string{"red", "green", "blue", "--"}, completeOK[args](t))
}

func TestCompleteEnumFlag(t *testing.T) {
	type args struct {
		color EnumArg[color] `long:"color" help:"tint"`
	}
	got := completeOK[args](t, "--color", "")
	assert.ElementsMatch(t, []string{"red", "green", "blue"}, got)
}

func TestCompleteEnumFlagEq(t *testing.T) {
	type args struct {
		color EnumArg[color] `long:"color"`
	}
	assert.Equal(t, []string{"--color=green"}, completeOK[args](t, "--color=g"))
}

func TestCompleteEnumFlagEqAll(t *testing.T) {
	type args struct {
		color EnumArg[color] `long:"color"`
	}
	got := completeOK[args](t, "--color=")
	assert.ElementsMatch(t, []string{"--color=blue", "--color=green", "--color=red"}, got)
}

func TestCompleteDashField(t *testing.T) {
	got := completeOK[dashArgs](t, "a", "")
	assert.Contains(t, got, "--")
}

func TestCompleteMidProductHidesFlags(t *testing.T) {
	type args struct {
		force Flag `long:"force"`
		pair  [2]StringArg
	}
	assert.Empty(t, completeOK[args](t, "name", ""))
	got := completeOK[args](t)
	assert.ElementsMatch(t, []string{"--force", "--"}, got)
}

func TestCompleteAfterValueFlag(t *testing.T) {
	type args struct {
		name  StringArg `long:"name"`
		force Flag      `long:"force"`
	}
	assert.Empty(t, completeOK[args](t, "--name", ""))
}

func TestCompleteOnceValueFlag(t *testing.T) {
	type args struct {
		name  StringArg `long:"name" short:"n"`
		force Flag      `long:"force"`
	}
	got := completeOK[args](t, "--name", "x", "")
	assert.ElementsMatch(t, []string{"--force", "--"}, got)
}

func TestCompleteOnceFlag(t *testing.T) {
	type args struct {
		force Flag      `long:"force" short:"f"`
		name  StringArg `long:"name"`
	}
	got := completeOK[args](t, "--force", "")
	assert.ElementsMatch(t, []string{"--name", "--"}, got)
}

func TestCompleteOnceShortHidesLong(t *testing.T) {
	type args struct {
		name  StringArg `long:"name" short:"n"`
		force Flag      `long:"force"`
	}
	got := completeOK[args](t, "-n", "x", "")
	assert.ElementsMatch(t, []string{"--force", "--"}, got)
}

func TestCompleteOnceParentFlag(t *testing.T) {
	got := completeOK[inheritApp](t, "add", "--force", "")
	assert.ElementsMatch(t, []string{"--name", "--verbose", "-v", "--label", "--"}, got)
}

func TestCompleteOptionalProductAllowsFlags(t *testing.T) {
	type args struct {
		force Flag `long:"force"`
		pair  KV[string, *StringArg]
	}
	got := completeOK[args](t, "name", "")
	assert.ElementsMatch(t, []string{"--force", "--"}, got)
}

func TestCompleteRepeatableFlag(t *testing.T) {
	got := completeOK[tagsArgs](t, "--tag", "a", "")
	assert.ElementsMatch(t, []string{"--tag", "-t", "--"}, got)
}

func TestCompleteCommandHelp(t *testing.T) {
	suggestions, err := Complete[listedApp]()
	require.NoError(t, err)
	var help string
	for _, suggestion := range suggestions {
		if suggestion.Text == "copy" {
			help = suggestion.Help
		}
	}
	assert.Equal(t, "copy files", help)
}

func TestCompleteSuggestionNotUnknown(t *testing.T) {
	suggestions, err := Complete[appArgs]()
	require.NoError(t, err)
	for _, suggestion := range suggestions {
		if suggestion.Kind == SuggestDash {
			continue
		}
		_, err := Parse[appArgs](suggestion.Text)
		if err != nil {
			assert.NotErrorIs(t, err, ErrUnknownFlag, suggestion.Text)
			assert.NotErrorIs(t, err, ErrUnknownCommand, suggestion.Text)
		}
	}
}

func TestCompleteNestedCommand(t *testing.T) {
	got := completeOK[nestedApp](t, "mid", "")
	assert.ElementsMatch(t, []string{"inner", "--"}, got)
	got = completeOK[nestedApp](t, "mid", "inner", "")
	assert.ElementsMatch(t, []string{"--name", "--"}, got)
}

func TestCompleteShortCluster(t *testing.T) {
	got := completeOK[appArgs](t, "-vv", "")
	assert.ElementsMatch(t, []string{"add", "rm", "--verbose", "-v", "--"}, got)
}

func TestCompleteAfterEqualsValue(t *testing.T) {
	got := completeOK[appArgs](t, "--verbose=2", "")
	assert.ElementsMatch(t, []string{"add", "rm", "--verbose", "-v", "--"}, got)
}

func TestCompleteAppFlags(t *testing.T) {
	got := completeOK[App[None]](t)
	assert.Subset(t, got, []string{"--verbose", "-v", "--help", "-h", "--version", "--profile-dir"})
}
