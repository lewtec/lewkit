package apk

import (
	"testing"

	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/require"
)

func TestLibsGraph(t *testing.T) {
	work := t.TempDir()
	mainDir := t.TempDir()
	graph, paths, cleanup, err := libsGraph(work, mainDir, []string{"arm64-v8a", "armeabi-v7a", "x86_64"}, version.Info{}, "br.tec.lew.demo", false)
	cleanup()
	require.NoError(t, err)
	require.Equal(t, []string{"jni"}, graph.Defaults)
	require.Len(t, paths, 3)

	by := map[string]workflow.Step{}
	for _, step := range graph.Steps {
		by[step.Name] = step
	}
	require.Equal(t, []string{"abi/arm64-v8a", "abi/armeabi-v7a", "abi/x86_64"}, by["jni"].Deps)
	require.Equal(t, "jni depends on every abi", by["jni"].Desc)
	require.Equal(t, taskgroup.Control, by["jni"].Pool)
	require.Empty(t, by["jni"].Tasks)

	require.Equal(t, []string{"toolexec"}, by["abi/arm64-v8a"].Deps)
	require.Equal(t, "arm64-v8a after toolexec", by["abi/arm64-v8a"].Desc)
	require.True(t, by["abi/arm64-v8a"].Orchestrator)
	require.True(t, by["abi/armeabi-v7a"].Orchestrator)
	require.True(t, by["abi/x86_64"].Orchestrator)
	require.Empty(t, by["abi/armeabi-v7a"].Deps)
	require.Empty(t, by["abi/x86_64"].Deps)
	require.True(t, by["toolexec"].Orchestrator)
	require.Empty(t, by["toolexec"].Deps)

	graph, _, cleanup, err = libsGraph(work, mainDir, []string{"arm64-v8a", "x86_64"}, version.Info{}, "br.tec.lew.demo", true)
	cleanup()
	require.NoError(t, err)
	by = map[string]workflow.Step{}
	for _, step := range graph.Steps {
		by[step.Name] = step
	}
	_, ok := by["toolexec"]
	require.False(t, ok)
	require.Empty(t, by["abi/arm64-v8a"].Deps)
	require.Equal(t, "arm64-v8a", by["abi/arm64-v8a"].Desc)
	require.Equal(t, []string{"abi/arm64-v8a", "abi/x86_64"}, by["jni"].Deps)

	_, _, cleanup, err = libsGraph(work, mainDir, []string{"mips"}, version.Info{}, "br.tec.lew.demo", false)
	cleanup()
	require.ErrorIs(t, err, ErrUnsupportedABI)
}

func TestLibsGraphDefaultIsArm64(t *testing.T) {
	graph, _, cleanup, err := libsGraph(t.TempDir(), t.TempDir(), nil, version.Info{}, "br.tec.lew.demo", false)
	cleanup()
	require.NoError(t, err)
	require.Equal(t, []string{"toolexec", "abi/arm64-v8a", "jni"}, stepNames(graph))
	require.Equal(t, []string{"toolexec"}, graph.Steps[1].Deps)
	require.Equal(t, []string{"abi/arm64-v8a"}, graph.Steps[2].Deps)
}

func stepNames(graph workflow.Graph) []string {
	names := make([]string, len(graph.Steps))
	for i, step := range graph.Steps {
		names[i] = step.Name
	}
	return names
}
