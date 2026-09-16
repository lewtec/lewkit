package progress

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func TestTickRefreshKeepsNodesOnModel(t *testing.T) {
	s, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	t.Cleanup(func() { _ = s.Wait() })
	block := make(chan struct{})
	taskgroup.Go(ctx, "hold", taskgroup.CPU, func(context.Context, *taskgroup.Status) error {
		<-block
		return nil
	})
	t.Cleanup(func() { close(block) })

	m := newModel(s)
	deadline := time.Now().Add(time.Second)
	var got model
	for {
		next, _ := m.Update(tickMsg{})
		got = next.(model)
		if len(got.nodes) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("tick did not copy List onto the model")
		}
		m = got
	}
	if got.nodes[0].Name != "hold" {
		t.Fatalf("nodes = %#v, want hold", got.nodes)
	}
}

func TestLayoutTreePrefixes(t *testing.T) {
	rows := layout([]taskgroup.Node{
		{ID: 1, Name: "bundle", State: taskgroup.Running, LiveChildren: 2},
		{ID: 2, Parent: 1, Name: "icons", State: taskgroup.Running, LiveChildren: 1},
		{ID: 3, Parent: 2, Name: "png", State: taskgroup.Pending},
		{ID: 4, Parent: 1, Name: "manifest", State: taskgroup.Pending},
		{ID: 5, Name: "lint", State: taskgroup.Done},
	})
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
	if rows[0].tree != "" {
		t.Errorf("root tree = %q, want empty", rows[0].tree)
	}
	if rows[1].tree != "├ " {
		t.Errorf("icons tree = %q, want ├ ", rows[1].tree)
	}
	if rows[2].tree != "│ └ " {
		t.Errorf("png tree = %q, want │ └ ", rows[2].tree)
	}
	if rows[3].tree != "└ " {
		t.Errorf("manifest tree = %q, want └ ", rows[3].tree)
	}
	if rows[4].tree != "" {
		t.Errorf("lint tree = %q, want empty", rows[4].tree)
	}
}

func TestLayoutHiddenChildren(t *testing.T) {
	rows := layout([]taskgroup.Node{
		{ID: 1, Name: "many", LiveChildren: 10},
		{ID: 2, Parent: 1, Name: "cpu:0"},
		{ID: 3, Parent: 1, Name: "cpu:1"},
	})
	if rows[0].hidden != 8 {
		t.Fatalf("hidden = %d, want 8", rows[0].hidden)
	}
}

func TestFormatRowTreeAndBar(t *testing.T) {
	line := formatRow(treeRow{
		node: taskgroup.Node{
			Name:    "build",
			Pool:    taskgroup.CPU,
			State:   taskgroup.Running,
			Message: "part 1/4",
			Current: 1,
			Total:   4,
		},
		tree: "├ ",
	}, 80)
	if !strings.HasPrefix(line, "├ ▶ 🧠 build: part 1/4 [") {
		t.Fatalf("line = %q", line)
	}
	if !strings.HasSuffix(line, "]") {
		t.Fatalf("line = %q, want trailing ]", line)
	}
	if cellWidth(line) != 80 {
		t.Fatalf("width = %d, want 80 (%q)", cellWidth(line), line)
	}
}

func TestFormatRowPendingNoBar(t *testing.T) {
	line := formatRow(treeRow{
		node: taskgroup.Node{Name: "install", Pool: taskgroup.IO, State: taskgroup.Pending},
		tree: "└ ",
	}, 80)
	if !strings.HasPrefix(line, "└ ⏸ 💾 install") {
		t.Fatalf("line = %q", line)
	}
	if strings.Contains(line, "[") {
		t.Fatalf("pending kept a bar: %q", line)
	}
}

func TestCancelDoesNotQuitUntilEmpty(t *testing.T) {
	s, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	t.Cleanup(func() { _ = s.Wait() })
	block := make(chan struct{})
	taskgroup.Go(ctx, "hold", taskgroup.CPU, func(ctx context.Context, _ *taskgroup.Status) error {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-block:
			return nil
		}
	})
	t.Cleanup(func() { close(block) })

	m := newModel(s)
	next, cmd := m.Update(cancelMsg{})
	got := next.(model)
	if got.shouldQuit() {
		t.Fatal("ctrl+c must not quit before List is empty")
	}
	if cmd != nil {
		t.Fatal("ctrl+c must not return a cmd")
	}
}

func TestQuitAfterDoneWhenListEmpty(t *testing.T) {
	m := newModel(nil)
	m.done = true
	next, cmd := m.Update(tickMsg{})
	got := next.(model)
	if !got.shouldQuit() {
		t.Fatal("want quit when done and list empty")
	}
	if cmd == nil {
		t.Fatal("want Quit cmd")
	}
}

func TestKeepTickingWhenDoneButLive(t *testing.T) {
	m := newModel(nil)
	m.done = true
	m.nodes = []taskgroup.Node{{ID: 1, Name: "left", State: taskgroup.Pending}}
	next, cmd := m.Update(tickMsg{})
	got := next.(model)
	if got.shouldQuit() {
		t.Fatal("must not quit while live rows remain")
	}
	if cmd == nil {
		t.Fatal("want another tick while live rows remain")
	}
}

func TestViewResizeSwitchesLayout(t *testing.T) {
	m := newModel(nil)
	m.sync([]taskgroup.Node{
		{ID: 1, Name: "build", Pool: taskgroup.CPU, State: taskgroup.Running, Message: "part 1/4", Current: 1, Total: 4},
	})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	m = next.(model)
	narrow := strings.TrimSuffix(m.View().Content, "\n")
	if !strings.Contains(narrow, "%") || strings.Contains(narrow, "[") {
		t.Fatalf("width 40 = %q, want percent layout", narrow)
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(model)
	wide := strings.TrimSuffix(m.View().Content, "\n")
	if !strings.Contains(wide, "[") || strings.Contains(wide, "%") {
		t.Fatalf("width 80 = %q, want expanding bar", wide)
	}
}
