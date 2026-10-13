package main

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/require"
)

func TestSimGraph(t *testing.T) {
	graph := simGraph("/repo")
	require.Equal(t, []string{"sim"}, graph.Defaults)
	require.Equal(t, "/repo", graph.Dir)

	by := map[string]workflow.Step{}
	for _, step := range graph.Steps {
		by[step.Name] = step
		require.NotEmpty(t, step.Tasks)
	}
	require.Empty(t, by["build"].Deps)
	require.Empty(t, by["boot"].Deps)
	require.Equal(t, []string{"build", "boot"}, by["install"].Deps)
	require.Equal(t, []string{"install"}, by["launch"].Deps)
	require.Empty(t, by["ping"].Deps)
	require.Equal(t, []string{"launch"}, by["sim"].Deps)
	require.Contains(t, by["install"].Desc, "after build and boot")
	require.Contains(t, by["launch"].Desc, "after install")
	require.Contains(t, by["sim"].Desc, "after launch")
	require.Contains(t, by["ping"].Desc, "open app")
}

func TestIPhoneUDID(t *testing.T) {
	list := "    iPhone 16 (AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE) (Shutdown)\n    iPad (nope)\n"
	require.Equal(t, "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE", iphoneUDID(list))
	require.Empty(t, iphoneUDID("no phones"))
	require.True(t, strings.Contains(list, "iPhone"))
}

func TestTarget(t *testing.T) {
	app, err := cmd.Parse[cmd.App[simCmd]]()
	require.NoError(t, err)
	require.Equal(t, "sim", app.Args.target.Value().String())
	require.Equal(t, 0, app.Args.IO.Value())

	app, err = cmd.Parse[cmd.App[simCmd]]("--io", "2", "ping")
	require.NoError(t, err)
	require.Equal(t, "ping", app.Args.target.Value().String())
	require.Equal(t, 2, app.Args.IO.Value())

	_, err = cmd.Parse[cmd.App[simCmd]]("clean")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)

	_, err = cmd.Parse[cmd.App[simCmd]]("ping", "extra")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
}
