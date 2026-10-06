package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowUsage(t *testing.T) {
	text, err := cmd.Usage[workflowCmd](release.Name() + " workflow")
	require.NoError(t, err)
	assert.Contains(t, text, "make")
	assert.Contains(t, text, "ninja")
	assert.Contains(t, text, "makefile or a ninja file")
}

func TestWorkflowMakeHelp(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "workflow", "make", "--help")
	require.NotNil(t, app.Args.workflow)
	require.NotNil(t, app.Args.workflow.make)
}

func TestWorkflowMakeAndNinja(t *testing.T) {
	dir := t.TempDir()
	makefile := "all:\n\techo from-make > out.txt\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0o644))
	app := cmd.ParseOK[cmd.App[root]](t, "workflow", "make", "-C", dir, "all")
	require.NoError(t, app.Run(t.Context()))
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err)
	assert.Equal(t, "from-make\n", string(got))

	ninjaFile := "rule touch\n  command = echo from-ninja > $out\nbuild out2.txt: touch\ndefault out2.txt\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "build.ninja"), []byte(ninjaFile), 0o644))
	app = cmd.ParseOK[cmd.App[root]](t, "workflow", "ninja", "-C", dir)
	require.NoError(t, app.Run(t.Context()))
	got, err = os.ReadFile(filepath.Join(dir, "out2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "from-ninja\n", string(got))
}

func TestWorkflowMakeLoadIsScheduled(t *testing.T) {
	dir := t.TempDir()
	body := "$(shell while [ ! -f gate ]; do sleep 0.02; done)\nall:\n\techo hi > out.txt\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644))
	gate := filepath.Join(dir, "gate")
	t.Cleanup(func() { _ = os.WriteFile(gate, []byte("x"), 0o644) })

	app := cmd.ParseOK[cmd.App[root]](t, "workflow", "make", "-C", dir, "all")
	sess, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	errCh := make(chan error, 1)
	go func() { errCh <- app.Run(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	saw := false
	for time.Now().Before(deadline) {
		for _, n := range sess.List(32) {
			if n.Name == "make" {
				saw = true
			}
		}
		if saw {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	require.NoError(t, os.WriteFile(gate, []byte("x"), 0o644))
	require.True(t, saw, "makefile load was not a progress task")
	require.NoError(t, <-errCh)
	require.NoError(t, sess.Wait())
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hi\n", string(got))
}

func TestWorkflowRecipeStaysUnderMake(t *testing.T) {
	dir := t.TempDir()
	body := "all:\n\twhile [ ! -f gate ]; do sleep 0.02; done\n\techo hi > out.txt\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644))
	gate := filepath.Join(dir, "gate")
	t.Cleanup(func() { _ = os.WriteFile(gate, []byte("x"), 0o644) })

	app := cmd.ParseOK[cmd.App[root]](t, "workflow", "make", "-C", dir, "all")
	sess, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	errCh := make(chan error, 1)
	go func() { errCh <- app.Run(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	var parent taskgroup.ID
	child := false
	for time.Now().Before(deadline) && !child {
		nodes := sess.List(32)
		for _, n := range nodes {
			if n.Name == "make" {
				parent = n.ID
			}
		}
		for _, n := range nodes {
			if parent != 0 && n.Name == "all" && n.Parent == parent {
				child = true
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	require.NoError(t, os.WriteFile(gate, []byte("x"), 0o644))
	require.True(t, child, "recipe step was not a child of the make task")
	require.NoError(t, <-errCh)
	require.NoError(t, sess.Wait())
	got, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hi\n", string(got))
}

func TestWorkflowMakeAssign(t *testing.T) {
	dir := t.TempDir()
	body := "WHO = file\nall:\n\techo $(WHO) > who.txt\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"), []byte(body), 0o644))
	app := cmd.ParseOK[cmd.App[root]](t, "workflow", "make", "-C", dir, "WHO=cli")
	require.NoError(t, app.Run(t.Context()))
	got, err := os.ReadFile(filepath.Join(dir, "who.txt"))
	require.NoError(t, err)
	assert.Equal(t, "cli\n", string(got))
}
