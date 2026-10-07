package workflow

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunSkipsFreshOutputs(t *testing.T) {
	dir := t.TempDir()
	g := Graph{
		Dir:      dir,
		Defaults: []string{"a"},
		Steps: []Step{{
			Name:    "a",
			Outputs: []string{"a.txt"},
			Tasks: []Task{
				Command{Text: "echo a > a.txt"},
				Command{Text: "echo ran >> runs"},
			},
		}},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	require.NoError(t, Run(t.Context(), g, nil))
	got, err := os.ReadFile(filepath.Join(dir, "runs"))
	require.NoError(t, err)
	assert.Equal(t, "ran\n", string(got))
	body, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "a\n", string(body))
}

func TestRunRebuildsWhenDependencyRuns(t *testing.T) {
	dir := t.TempDir()
	g := Graph{
		Dir:      dir,
		Defaults: []string{"b"},
		Steps: []Step{
			{
				Name:  "a",
				Phony: true,
				Tasks: []Task{Command{Text: "echo a > a.txt"}},
			},
			{
				Name:    "b",
				Deps:    []string{"a"},
				Inputs:  []string{"a.txt"},
				Outputs: []string{"b.txt"},
				Tasks:   []Task{Command{Text: "cat a.txt > b.txt; echo ran >> runs"}},
			},
		},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	require.NoError(t, Run(t.Context(), g, nil))
	got, err := os.ReadFile(filepath.Join(dir, "runs"))
	require.NoError(t, err)
	assert.Equal(t, "ran\nran\n", string(got))
}

func TestRunStopsDependentsAfterFailure(t *testing.T) {
	dir := t.TempDir()
	g := Graph{
		Dir:      dir,
		Defaults: []string{"next"},
		Steps: []Step{
			{Name: "bad", Tasks: []Task{Command{Text: "exit 1"}}},
			{
				Name:    "next",
				Deps:    []string{"bad"},
				Outputs: []string{"no.txt"},
				Tasks:   []Task{Command{Text: "echo no > no.txt"}},
			},
		},
	}
	err := Run(t.Context(), g, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
	_, statErr := os.Stat(filepath.Join(dir, "no.txt"))
	assert.Error(t, statErr)
}

func TestRunCycleAndUnknown(t *testing.T) {
	dir := t.TempDir()
	cycle := Graph{Dir: dir, Steps: []Step{
		{Name: "a", Deps: []string{"b"}},
		{Name: "b", Deps: []string{"a"}},
	}}
	require.ErrorIs(t, Run(t.Context(), cycle, []string{"a"}), ErrCycle)
	unknown := Graph{Dir: dir, Steps: []Step{{Name: "a", Deps: []string{"missing"}}}}
	require.ErrorIs(t, Run(t.Context(), unknown, []string{"a"}), ErrUnknownStep)
	require.ErrorIs(t, Run(t.Context(), Graph{}, nil), ErrNoTarget)
	require.Error(t, Run(nil, Graph{}, nil))
}

func TestRunAlias(t *testing.T) {
	dir := t.TempDir()
	g := Graph{
		Dir:   dir,
		Alias: map[string]string{"extra": "main"},
		Steps: []Step{{
			Name:    "main",
			Outputs: []string{"main.txt", "extra.txt"},
			Tasks:   []Task{Command{Text: "echo m > main.txt; echo e > extra.txt"}},
		}},
	}
	require.NoError(t, Run(t.Context(), g, []string{"extra"}))
	got, err := os.ReadFile(filepath.Join(dir, "extra.txt"))
	require.NoError(t, err)
	assert.Equal(t, "e\n", string(got))
}

func TestRunParallelJoin(t *testing.T) {
	dir := t.TempDir()
	g := Graph{
		Dir:      dir,
		Defaults: []string{"join"},
		Steps: []Step{
			{Name: "left", Outputs: []string{"left.txt"}, Tasks: []Task{Command{Text: "echo L > left.txt"}}},
			{Name: "right", Outputs: []string{"right.txt"}, Tasks: []Task{Command{Text: "echo R > right.txt"}}},
			{
				Name:    "join",
				Deps:    []string{"left", "right"},
				Inputs:  []string{"left.txt", "right.txt"},
				Outputs: []string{"join.txt"},
				Tasks:   []Task{Command{Text: "cat left.txt right.txt > join.txt"}},
			},
		},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	got, err := os.ReadFile(filepath.Join(dir, "join.txt"))
	require.NoError(t, err)
	assert.Equal(t, "L\nR\n", string(got))
	require.NoError(t, Run(t.Context(), g, nil))
	got, err = os.ReadFile(filepath.Join(dir, "join.txt"))
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(got), "ran"))
}

func TestJoinSlashName(t *testing.T) {
	assert.Equal(t, "/tmp/work/src/a.c", Join("/tmp/work", "src/a.c"))
	assert.Equal(t, "/tmp/work/a.c", Join("/tmp/work", "src/../a.c"))
	assert.Equal(t, "/abs/a.c", Join("/tmp/work", "/abs/a.c"))
	assert.Equal(t, "/tmp/work", Join("/tmp/work", "."))
	assert.Equal(t, "", Join("/tmp/work", ""))
	assert.Equal(t, "src/a.c", Join("", "src/a.c"))
	assert.Equal(t, "/tmp/work/src", outputParent("/tmp/work", "src/a.c"))
	assert.Equal(t, "/tmp/work", outputParent("/tmp/work", "a.c"))
	assert.Equal(t, "/abs", outputParent("/tmp/work", "/abs/a.c"))
}

func TestMergeEnvReplaces(t *testing.T) {
	got := mergeEnv([]string{"A=1", "B=2"}, []string{"B=3", "C=4"})
	assert.Equal(t, []string{"A=1", "B=3", "C=4"}, got)
}

func TestRunInProcess(t *testing.T) {
	dir := t.TempDir()
	var n int
	var saw string
	g := Graph{
		Dir:      dir,
		Defaults: []string{"out"},
		Steps: []Step{{
			Name:    "out",
			Outputs: []string{"out.txt"},
			Tasks: []Task{Func(func(ctx context.Context, st *taskgroup.Status) error {
				n++
				saw = Dir(ctx)
				st.Update("write")
				return os.WriteFile(filepath.Join(dir, "out.txt"), []byte("ok\n"), 0o644)
			})},
		}},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	require.NoError(t, Run(t.Context(), g, nil))
	assert.Equal(t, 1, n)
	assert.Equal(t, dir, saw)
	body, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	require.NoError(t, err)
	assert.Equal(t, "ok\n", string(body))
}

func TestRunInProcessEveryTime(t *testing.T) {
	var n int
	g := Graph{
		Defaults: []string{"tick"},
		Steps: []Step{{
			Name: "tick",
			Tasks: []Task{Func(func(context.Context, *taskgroup.Status) error {
				n++
				return nil
			})},
		}},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	require.NoError(t, Run(t.Context(), g, nil))
	assert.Equal(t, 2, n)
}

func TestRunInProcessStopsDependents(t *testing.T) {
	var n int
	g := Graph{
		Defaults: []string{"next"},
		Steps: []Step{
			{Name: "bad", Tasks: []Task{Func(func(context.Context, *taskgroup.Status) error {
				return errors.New("boom")
			})}},
			{Name: "next", Deps: []string{"bad"}, Tasks: []Task{Func(func(context.Context, *taskgroup.Status) error {
				n++
				return nil
			})}},
		},
	}
	err := Run(t.Context(), g, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
	assert.Equal(t, 0, n)
}

func TestRunMixesFuncAndCommand(t *testing.T) {
	dir := t.TempDir()
	g := Graph{
		Dir:      dir,
		Defaults: []string{"copy"},
		Steps: []Step{
			{
				Name:    "src",
				Outputs: []string{"src.txt"},
				Tasks: []Task{Func(func(context.Context, *taskgroup.Status) error {
					return os.WriteFile(filepath.Join(dir, "src.txt"), []byte("in\n"), 0o644)
				})},
			},
			{
				Name:    "copy",
				Deps:    []string{"src"},
				Inputs:  []string{"src.txt"},
				Outputs: []string{"copy.txt"},
				Tasks:   []Task{Command{Text: "cat src.txt > copy.txt"}},
			},
		},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	got, err := os.ReadFile(filepath.Join(dir, "copy.txt"))
	require.NoError(t, err)
	assert.Equal(t, "in\n", string(got))
}

func TestRunDownloadsAndExtracts(t *testing.T) {
	body := []byte("payload\n")
	archive := zipBytes(t, "pkg/hello.txt", body)
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	g := Graph{
		Dir:      dir,
		Defaults: []string{"unpack"},
		Steps: []Step{
			{
				Name:    "fetch",
				Outputs: []string{"vendor/pkg.zip"},
				Tasks: []Task{Download{
					URL:  server.URL + "/pkg.zip",
					Dest: "vendor/pkg.zip",
				}},
			},
			{
				Name:    "unpack",
				Deps:    []string{"fetch"},
				Inputs:  []string{"vendor/pkg.zip"},
				Outputs: []string{"out/hello.txt"},
				Tasks:   []Task{Extract{Source: "vendor/pkg.zip", Dest: "out"}},
			},
		},
	}
	require.NoError(t, Run(t.Context(), g, nil))
	require.NoError(t, Run(t.Context(), g, nil))
	assert.Equal(t, int32(1), hits.Load())
	got, err := os.ReadFile(filepath.Join(dir, "out", "hello.txt"))
	require.NoError(t, err)
	assert.Equal(t, body, got)
}

func TestRunDownloadRejectsHash(t *testing.T) {
	t.Setenv("FETCHURL_SERVER", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	var n int
	g := Graph{
		Dir:      dir,
		Defaults: []string{"next"},
		Steps: []Step{
			{
				Name:    "fetch",
				Outputs: []string{"hello.txt"},
				Tasks: []Task{Download{
					URL:     server.URL,
					Dest:    "hello.txt",
					Options: tool.DownloadOptions{Hash: "sha256:deadbeef"},
				}},
			},
			{
				Name:  "next",
				Deps:  []string{"fetch"},
				Tasks: []Task{Func(func(context.Context, *taskgroup.Status) error { n++; return nil })},
			},
		},
	}
	err := Run(t.Context(), g, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch")
	assert.Equal(t, 0, n)
	_, statErr := os.Stat(filepath.Join(dir, "hello.txt"))
	assert.Error(t, statErr)
}

func zipBytes(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entry, err := zw.Create(name)
	require.NoError(t, err)
	_, err = entry.Write(body)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}
