package term

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionWritersAndWatch(t *testing.T) {
	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)

	first, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 10, Cols: 20})
	require.NoError(t, err)
	second, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 10, Cols: 20})
	require.NoError(t, err)
	assert.Empty(t, first.LastWriter)
	assert.NotEqual(t, first.ID, second.ID)

	require.NoError(t, session.Write(ctx, WriteRequest{
		ID: first.ID, Data: "x", Rows: 12, Cols: 40, Client: "writer-a",
	}))
	got, err := session.Pane(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, 12, got.Rows)
	assert.Equal(t, 40, got.Cols)
	assert.Equal(t, "writer-a", got.LastWriter)

	other, err := session.Pane(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, 10, other.Rows)
	assert.Equal(t, 20, other.Cols)
	assert.Empty(t, other.LastWriter)

	again, err := session.Pane(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, got.Rows, again.Rows)
	assert.Equal(t, got.Cols, again.Cols)
	assert.Equal(t, got.LastWriter, again.LastWriter)

	err = session.Write(ctx, WriteRequest{ID: first.ID, Data: "z", Rows: 12, Cols: 0})
	require.ErrorIs(t, err, ErrSize)
	err = session.Write(ctx, WriteRequest{ID: first.ID, Data: "z", Rows: -1, Cols: -1})
	require.ErrorIs(t, err, ErrSize)

	require.NoError(t, session.Write(ctx, WriteRequest{ID: first.ID, Data: "y", Client: "reader"}))
	kept, err := session.Pane(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, 12, kept.Rows)
	assert.Equal(t, 40, kept.Cols)
	assert.Equal(t, "writer-a", kept.LastWriter)

	require.NoError(t, session.Write(ctx, WriteRequest{
		ID: second.ID, Data: "q", Rows: 30, Cols: 100, Client: "writer-b",
	}))
	moved, err := session.Pane(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, 30, moved.Rows)
	assert.Equal(t, 100, moved.Cols)
	assert.Equal(t, "writer-b", moved.LastWriter)
	untouched, err := session.Pane(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, 12, untouched.Rows)
	assert.Equal(t, 40, untouched.Cols)

	filtered := make(chan Event, 8)
	cancel, err := session.Watch([]string{first.ID}, false, "", func(ev Event) {
		filtered <- ev
	})
	require.NoError(t, err)
	t.Cleanup(cancel)
	sawInitial := false
	select {
	case ev := <-filtered:
		require.NotNil(t, ev.Pane)
		assert.Equal(t, first.ID, ev.Pane.ID)
		sawInitial = true
	case <-time.After(time.Second):
	}
	require.True(t, sawInitial)

	third, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 8, Cols: 8})
	require.NoError(t, err)
	leaked := ""
	for {
		select {
		case ev := <-filtered:
			if ev.Notice == NoticeSession {
				leaked = "session"
				continue
			}
			if ev.Pane != nil && ev.Pane.ID != first.ID {
				leaked = ev.Pane.ID
			}
		default:
			goto watched
		}
	}
watched:
	assert.Empty(t, leaked)
	assert.NotEqual(t, first.ID, third.ID)

	require.NoError(t, session.Kill(ctx, first.ID))
	require.Eventually(t, func() bool {
		pane, err := session.Pane(ctx, first.ID)
		return err == nil && !pane.Alive
	}, 2*time.Second, 20*time.Millisecond)
}

func TestSessionTree(t *testing.T) {
	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)

	editor, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 8, Cols: 8, Path: "work/"})
	require.NoError(t, err)
	assert.Equal(t, "work/"+editor.ID, editor.Path)

	notes, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 8, Cols: 8, Path: "notes"})
	require.NoError(t, err)
	assert.Equal(t, "notes", notes.Path)

	root, err := session.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, root.Entries, 2)
	assert.Equal(t, "notes", root.Entries[0].Path)
	assert.Equal(t, notes.ID, root.Entries[0].Pane)
	assert.True(t, root.Entries[1].Dir)
	assert.Equal(t, "work", root.Entries[1].Path)

	inside, err := session.List(ctx, "work")
	require.NoError(t, err)
	require.Len(t, inside.Entries, 1)
	assert.Equal(t, editor.ID, inside.Entries[0].Pane)

	_, err = session.List(ctx, "notes")
	require.ErrorIs(t, err, ErrPath)
	_, err = session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 4, Cols: 4, Path: "work"})
	require.ErrorIs(t, err, ErrPath)

	require.NoError(t, session.Move(ctx, editor.ID, "work/editor"))
	moved, err := session.Pane(ctx, editor.ID)
	require.NoError(t, err)
	assert.Equal(t, "work/editor", moved.Path)

	seen := make(chan Event, 8)
	cancel, err := session.Watch(nil, false, "work", func(ev Event) {
		seen <- ev
	})
	require.NoError(t, err)
	t.Cleanup(cancel)
	require.Eventually(t, func() bool {
		select {
		case ev := <-seen:
			return ev.Notice == NoticeSession
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	other, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 4, Cols: 4, Path: "elsewhere"})
	require.NoError(t, err)
	child, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 4, Cols: 4, Path: "work/child"})
	require.NoError(t, err)

	var panes []string
	require.Eventually(t, func() bool {
		select {
		case ev := <-seen:
			if ev.Notice == NoticePane && ev.Pane != nil {
				panes = append(panes, ev.Pane.ID)
			}
			return contains(panes, child.ID)
		default:
			return contains(panes, child.ID)
		}
	}, time.Second, 10*time.Millisecond)
	assert.NotContains(t, panes, other.ID)

	info, err := session.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(ProtocolLevel), info.Level)
	assert.NotContains(t, info.Dirs, "work")
	require.NoError(t, session.CreateDir(ctx, "work/extra"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Dirs, "work/extra")
	assert.NotContains(t, info.Dirs, "work")
	err = session.CreateDir(ctx, "notes")
	require.ErrorIs(t, err, ErrPath)
}

func TestSessionStoresAnEmptyDirectory(t *testing.T) {
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)

	require.NoError(t, session.CreateDir(ctx, "pockets"))
	info, err := session.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(ProtocolLevel), info.Level)
	assert.Equal(t, []string{"pockets"}, info.Dirs)

	root, err := session.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, root.Entries, 1)
	assert.True(t, root.Entries[0].Dir)
	assert.Equal(t, "pockets", root.Entries[0].Path)

	require.NoError(t, session.CreateDir(ctx, "pockets"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"pockets"}, info.Dirs)

	require.NoError(t, session.CreateDir(ctx, "pockets/bin"))
	require.NoError(t, session.CreateDir(ctx, "archive"))
	require.NoError(t, session.MoveDir(ctx, "pockets", "archive/pockets"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"archive", "archive/pockets", "archive/pockets/bin"}, info.Dirs)

	require.ErrorIs(t, session.MoveDir(ctx, "archive/pockets", "archive/pockets/bin/nope"), ErrPath)
	require.ErrorIs(t, session.RemoveDir(ctx, "missing"), ErrPath)

	require.NoError(t, session.RemoveDir(ctx, "archive/pockets"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"archive"}, info.Dirs)

	root, err = session.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, root.Entries, 1)
	assert.Equal(t, "archive", root.Entries[0].Path)
}

func TestSessionFactsFollowThePane(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "term.state.json")
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	session.useState(statePath)

	session.mu.Lock()
	session.panes["ed"] = &live{
		info: PaneInfo{ID: "ed", Path: "work/editor", Dir: "/old", Alive: true, Exit: -1, Rows: 2, Cols: 40},
		grid: NewGrid(2, 40),
	}
	session.paths["work/editor"] = "ed"
	session.order = []string{"ed"}
	session.mu.Unlock()

	session.feed("ed", []byte("hello\x1b]7;file://host/new\x1b\\\x1b]777;set;task;parser\x07\x1b]777;set;progress;halfway\x07\x1b]777;set;state;idle\x07"))
	pane, err := session.Pane(ctx, "ed")
	require.NoError(t, err)
	assert.Equal(t, "hello", pane.Line(0))
	assert.Equal(t, "/new", pane.Cwd)
	assert.Equal(t, "/old", pane.Dir)
	info, err := session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "task", Value: "parser"})
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "progress", Value: "halfway"})
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "cwd", Value: "/new"})

	session.feed("ed", []byte("\x1b]777;cwd;/after\x07\x1b]777;del;progress\x07"))
	pane, err = session.Pane(ctx, "ed")
	require.NoError(t, err)
	assert.Equal(t, "/after", pane.Cwd)
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "cwd", Value: "/after"})
	assert.NotContains(t, info.Facts, Fact{Path: "ed", Key: "progress", Value: "halfway"})

	blob, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var saved termState
	require.NoError(t, json.Unmarshal(blob, &saved))
	assert.Equal(t, "parser", saved.Facts["ed"]["task"])
	assert.Equal(t, "/after", saved.Facts["ed"]["cwd"])
	assert.Empty(t, saved.Facts["ed"]["progress"])

	require.NoError(t, session.CreateDir(ctx, "pockets"))
	session.mu.Lock()
	session.panes["page"] = &live{
		info: PaneInfo{ID: "page", Path: "pockets/work/editor", Dir: "/old", Alive: true, Exit: -1, Rows: 2, Cols: 20},
		grid: NewGrid(2, 20),
	}
	session.paths["pockets/work/editor"] = "page"
	session.order = append(session.order, "page")
	session.mu.Unlock()
	session.feed("page", []byte("\x1b]777;set;task;nested\x07"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "page", Key: "task", Value: "nested"})

	require.NoError(t, session.Remove(ctx, "ed"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.NotContains(t, info.Panes, "ed")
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "task", Value: "parser"})

	again, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(again.Close)
	again.useState(statePath)
	info, err = again.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "task", Value: "parser"})
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "cwd", Value: "/after"})
	assert.Contains(t, info.Facts, Fact{Path: "page", Key: "task", Value: "nested"})

	bad := filepath.Join(t.TempDir(), "term.state.json")
	require.NoError(t, os.WriteFile(bad, []byte("nope"), 0o600))
	again.useState(bad)
	info, err = again.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "ed", Key: "task", Value: "parser"})

	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		return
	}
	spawn := t.TempDir()
	made, err := session.Create(ctx, CreateRequest{
		Command: []string{"/bin/cat"},
		Dir:     spawn,
		Rows:    4,
		Cols:    20,
		Path:    "work/next",
	})
	require.NoError(t, err)
	assert.Empty(t, made.Cwd)
	assert.Equal(t, spawn, made.Dir)

	session.mu.Lock()
	ok := session.setFact("work", "cwd", "/from-dir")
	session.mu.Unlock()
	require.True(t, ok)
	later, err := session.Create(ctx, CreateRequest{
		Command: []string{"/bin/cat"},
		Dir:     spawn,
		Rows:    4,
		Cols:    20,
		Path:    "work/later",
	})
	require.NoError(t, err)
	assert.Equal(t, "/from-dir", later.Cwd)
}

func TestCwdFollowsAQuietProcess(t *testing.T) {
	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	start := t.TempDir()
	next := t.TempDir()
	want, err := filepath.EvalSymlinks(next)
	require.NoError(t, err)
	quiet, err := session.Create(ctx, CreateRequest{
		Command: []string{"/bin/sh", "-c", "cd " + shellQuote(next) + "; exec /bin/sleep 30"},
		Dir:     start,
		Rows:    8,
		Cols:    40,
		Path:    "work/quiet",
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		info, err := session.Info(ctx)
		if err != nil {
			return false
		}
		for _, fact := range info.Facts {
			if fact.Path != quiet.ID || fact.Key != "cwd" {
				continue
			}
			got, err := filepath.EvalSymlinks(fact.Value)
			return err == nil && got == want
		}
		return false
	}, 3*time.Second, 20*time.Millisecond)
	panes, err := session.Panes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, panes)
	assert.Equal(t, start, panes[0].Dir)
	got, err := filepath.EvalSymlinks(panes[0].Cwd)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

func TestShellCwdFollowsTheProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		test.Skipf(t, "proc cwd: %s", runtime.GOOS)
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	got, ok := processCwd(os.Getpid())
	require.True(t, ok)
	wd, err = filepath.EvalSymlinks(wd)
	require.NoError(t, err)
	got, err = filepath.EvalSymlinks(got)
	require.NoError(t, err)
	assert.Equal(t, wd, got)

	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	start := t.TempDir()
	next := t.TempDir()
	pane, err := session.Create(ctx, CreateRequest{
		Command: []string{"/bin/sh"},
		Dir:     start,
		Rows:    12,
		Cols:    80,
		Path:    "work/shell",
	})
	require.NoError(t, err)
	wantStart, err := filepath.EvalSymlinks(start)
	require.NoError(t, err)
	wantNext, err := filepath.EvalSymlinks(next)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, err := session.Pane(ctx, pane.ID)
		if err != nil || got.Cwd == "" {
			return false
		}
		reported, err := filepath.EvalSymlinks(got.Cwd)
		return err == nil && reported == wantStart && got.Dir == start
	}, 3*time.Second, 20*time.Millisecond)
	require.NoError(t, session.Write(ctx, WriteRequest{ID: pane.ID, Data: "cd " + next + "\n"}))
	require.Eventually(t, func() bool {
		got, err := session.Pane(ctx, pane.ID)
		if err != nil || got.Cwd == "" {
			return false
		}
		reported, err := filepath.EvalSymlinks(got.Cwd)
		return err == nil && reported == wantNext && got.Dir == start
	}, 3*time.Second, 20*time.Millisecond)
	info, err := session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: pane.ID, Key: "cwd", Value: wantNext})
}

func TestFactsInheritLikeDirectories(t *testing.T) {
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)
	require.NoError(t, session.CreateDir(ctx, "pockets"))

	session.mu.Lock()
	session.panes["page"] = &live{
		info: PaneInfo{ID: "page", Path: "pockets/work/editor", Alive: true, Exit: -1, Rows: 2, Cols: 20},
		grid: NewGrid(2, 20),
	}
	session.paths["pockets/work/editor"] = "page"
	session.order = []string{"page"}
	okState := session.setFact("pockets", "state", "shared")
	okCwd := session.setFact("pockets", "cwd", "/pocket")
	okSibling := session.setFact("page", "cwd", "/sibling")
	session.mu.Unlock()
	require.True(t, okState)
	require.True(t, okCwd)
	require.True(t, okSibling)

	assert.Empty(t, session.savedCwd("archive/next"))
	assert.Equal(t, "/pocket", session.savedCwd("pockets/work/next"))
	assert.Equal(t, "shared", session.factAt("page", "pockets/work/editor", "state"))

	session.mu.Lock()
	okBlank := session.setFact("page", "state", "   ")
	session.mu.Unlock()
	require.True(t, okBlank)
	assert.Equal(t, "shared", session.factAt("page", "pockets/work/editor", "state"))
	session.mu.Lock()
	okDel := session.delFact("page", "state")
	session.mu.Unlock()
	require.True(t, okDel)

	session.feed("page", []byte("\x1b]777;set;task;nested\x07\x1b]777;set;state;mine\x07"))
	assert.Equal(t, "nested", session.factAt("page", "pockets/work/editor", "task"))
	assert.Equal(t, "mine", session.factAt("page", "pockets/work/editor", "state"))
	session.feed("page", []byte("\x1b]777;del;state\x07"))
	assert.Equal(t, "shared", session.factAt("page", "pockets/work/editor", "state"))

	require.NoError(t, session.Move(ctx, "page", "archive/editor"))
	assert.Equal(t, "", session.factAt("page", "archive/editor", "state"))
	assert.Equal(t, "nested", session.factAt("page", "archive/editor", "task"))
	info, err := session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "page", Key: "task", Value: "nested"})
	assert.Contains(t, info.Facts, Fact{Path: "pockets", Key: "state", Value: "shared"})

	session.mu.Lock()
	session.panes["box"] = &live{
		info: PaneInfo{ID: "box", Path: "loose", Alive: true, Exit: -1, Rows: 2, Cols: 20},
		grid: NewGrid(2, 20),
	}
	session.paths["loose"] = "box"
	session.order = append(session.order, "box")
	session.dirs["box"] = struct{}{}
	okPane := session.setFact("box", "task", "pane-local")
	okChild := session.setFact("box/bin", "state", "child")
	session.mu.Unlock()
	require.True(t, okPane)
	require.True(t, okChild)

	require.NoError(t, session.MoveDir(ctx, "pockets", "kept"))
	require.NoError(t, session.MoveDir(ctx, "box", "elsewhere"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.Contains(t, info.Facts, Fact{Path: "kept", Key: "state", Value: "shared"})
	assert.Contains(t, info.Facts, Fact{Path: "kept", Key: "cwd", Value: "/pocket"})
	assert.NotContains(t, info.Facts, Fact{Path: "pockets", Key: "state", Value: "shared"})
	assert.Contains(t, info.Facts, Fact{Path: "box", Key: "task", Value: "pane-local"})
	assert.Contains(t, info.Facts, Fact{Path: "elsewhere/bin", Key: "state", Value: "child"})
	assert.NotContains(t, info.Facts, Fact{Path: "box/bin", Key: "state", Value: "child"})
	assert.Equal(t, "nested", session.factAt("page", "archive/editor", "task"))
	assert.Contains(t, info.Dirs, "elsewhere")
	assert.NotContains(t, info.Dirs, "box")

	require.NoError(t, session.RemoveDir(ctx, "kept"))
	require.NoError(t, session.RemoveDir(ctx, "elsewhere"))
	info, err = session.Info(ctx)
	require.NoError(t, err)
	assert.NotContains(t, info.Facts, Fact{Path: "kept", Key: "state", Value: "shared"})
	assert.NotContains(t, info.Facts, Fact{Path: "elsewhere/bin", Key: "state", Value: "child"})
	assert.Contains(t, info.Facts, Fact{Path: "page", Key: "task", Value: "nested"})
	assert.Contains(t, info.Facts, Fact{Path: "box", Key: "task", Value: "pane-local"})

	legacyPath := filepath.Join(t.TempDir(), "term.state.json")
	require.NoError(t, os.WriteFile(legacyPath, []byte(`{"facts":{"work":{"task":"legacy"},"pockets/work":{"state":"old"}}}`), 0o600))
	legacy, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(legacy.Close)
	legacy.useState(legacyPath)
	assert.Equal(t, "legacy", legacy.factAt("ed", "work/editor", "task"))
	assert.Equal(t, "old", legacy.factAt("page", "pockets/work/editor", "state"))
	assert.Equal(t, "", legacy.factAt("ed", "work/editor", "state"))
}

func TestSessionRemoveDropsTheRecord(t *testing.T) {
	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)

	editor, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 8, Cols: 8, Path: "work/editor"})
	require.NoError(t, err)
	notes, err := session.Create(ctx, CreateRequest{Command: []string{"/bin/cat"}, Rows: 8, Cols: 8, Path: "notes"})
	require.NoError(t, err)

	stopped := make(chan struct{}, 1)
	session.stopWhenIdle(func() {
		select {
		case stopped <- struct{}{}:
		default:
		}
	})
	session.clientArrived()
	session.clientLeft()

	closed := make(chan string, 4)
	cancel, err := session.Watch([]string{notes.ID}, false, "", func(ev Event) {
		if ev.Notice == NoticePaneClosed && ev.Pane != nil {
			closed <- ev.Pane.ID
		}
	})
	require.NoError(t, err)
	t.Cleanup(cancel)

	require.NoError(t, session.Remove(ctx, notes.ID))
	_, err = session.Pane(ctx, notes.ID)
	require.ErrorIs(t, err, ErrPane)
	kept, err := session.Pane(ctx, editor.ID)
	require.NoError(t, err)
	assert.Equal(t, "work/editor", kept.Path)

	root, err := session.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, root.Entries, 1)
	assert.Equal(t, "work", root.Entries[0].Path)
	assert.True(t, root.Entries[0].Dir)

	require.Eventually(t, func() bool {
		select {
		case id := <-closed:
			return id == notes.ID
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	select {
	case <-stopped:
		require.Fail(t, "ubersession stopped while a pane was still open")
	default:
	}

	require.NoError(t, session.Remove(ctx, editor.ID))
	_, err = session.Pane(ctx, editor.ID)
	require.ErrorIs(t, err, ErrPane)
	require.Eventually(t, func() bool {
		select {
		case <-stopped:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func TestSessionPublishesInputModes(t *testing.T) {
	if _, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		test.Skipf(t, "pty: %v", err)
	}
	ctx := t.Context()
	session, err := Open(ctx)
	require.NoError(t, err)
	t.Cleanup(session.Close)

	pane, err := session.Create(ctx, CreateRequest{
		Command: []string{"/bin/sh", "-c", "printf '\\033[?1000;1002;1006h'"},
		Rows:    4,
		Cols:    20,
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, err := session.Pane(ctx, pane.ID)
		if err != nil {
			return false
		}
		return got.Input&InputMouse != 0 && got.Input&InputMotion != 0 && got.Input&InputSGR != 0
	}, 2*time.Second, 20*time.Millisecond)
}
