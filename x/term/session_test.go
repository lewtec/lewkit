package term

import (
	"os"
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
