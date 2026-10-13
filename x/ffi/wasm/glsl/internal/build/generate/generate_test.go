package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/require"
)

func TestGenerateArgs(t *testing.T) {
	app, err := cmd.Parse[cmd.App[generateCmd]]()
	require.NoError(t, err)
	require.Equal(t, 0, app.Args.CPU.Value())

	app, err = cmd.Parse[cmd.App[generateCmd]]("--cpu", "1")
	require.NoError(t, err)
	require.Equal(t, 1, app.Args.CPU.Value())

	_, err = cmd.Parse[cmd.App[generateCmd]]("extra")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
}

func TestWasmGraph(t *testing.T) {
	graph := wasmGraph("/src", "/work", "/out/glslang.wasm")
	require.Equal(t, []string{"glslang.wasm"}, graph.Defaults)
	require.Equal(t, "/src", graph.Dir)

	by := map[string][]string{}
	pool := map[string]taskgroup.PoolKind{}
	desc := map[string]string{}
	for _, step := range graph.Steps {
		by[step.Name] = step.Deps
		pool[step.Name] = step.Pool
		desc[step.Name] = step.Desc
		require.NotEmpty(t, step.Tasks)
	}
	require.Empty(t, by["tools"])
	require.Empty(t, by["source"])
	require.Equal(t, []string{"source", "tools"}, by["configure"])
	require.Equal(t, []string{"configure"}, by["libs"])
	require.Equal(t, []string{"libs"}, by["glslang.wasm"])
	require.Equal(t, taskgroup.Internet, pool["source"])
	require.Equal(t, taskgroup.IO, pool["configure"])
	require.Contains(t, desc["configure"], "after source and tools")
	require.Contains(t, desc["glslang.wasm"], "after libs")
	require.Contains(t, desc["libs"], "after configure")
}
