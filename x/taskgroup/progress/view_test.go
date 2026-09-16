package progress

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	var got model
	require.Eventually(t, func() bool {
		next, _ := m.Update(tickMsg{})
		got = next.(model)
		m = got
		return len(got.nodes) > 0
	}, time.Second, time.Millisecond)
	assert.Equal(t, "hold", got.nodes[0].Name)
}

func TestLayoutTreePrefixes(t *testing.T) {
	rows := layout([]taskgroup.Node{
		{ID: 1, Name: "bundle", State: taskgroup.Running, LiveChildren: 2},
		{ID: 2, Parent: 1, Name: "icons", State: taskgroup.Running, LiveChildren: 1},
		{ID: 3, Parent: 2, Name: "png", State: taskgroup.Pending},
		{ID: 4, Parent: 1, Name: "manifest", State: taskgroup.Pending},
		{ID: 5, Name: "lint", State: taskgroup.Done},
	})
	require.Len(t, rows, 5)
	assert.Empty(t, rows[0].tree)
	assert.Equal(t, "├ ", rows[1].tree)
	assert.Equal(t, "│ └ ", rows[2].tree)
	assert.Equal(t, "└ ", rows[3].tree)
	assert.Empty(t, rows[4].tree)
}

func TestLayoutHiddenChildren(t *testing.T) {
	rows := layout([]taskgroup.Node{
		{ID: 1, Name: "many", LiveChildren: 10},
		{ID: 2, Parent: 1, Name: "cpu:0"},
		{ID: 3, Parent: 1, Name: "cpu:1"},
	})
	assert.Equal(t, 8, rows[0].hidden)
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
	assert.True(t, strings.HasPrefix(line, "├ ▶ 🧠 build: part 1/4 ["), line)
	assert.True(t, strings.HasSuffix(line, "]"), line)
	assert.Equal(t, 80, cellWidth(line), line)
}

func TestFormatRowPendingNoBar(t *testing.T) {
	line := formatRow(treeRow{
		node: taskgroup.Node{Name: "install", Pool: taskgroup.IO, State: taskgroup.Pending},
		tree: "└ ",
	}, 80)
	assert.True(t, strings.HasPrefix(line, "└ ⏸ 💾 install"), line)
	assert.NotContains(t, line, "[")
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
	assert.False(t, got.shouldQuit())
	assert.Nil(t, cmd)
}

func TestQuitAfterDoneWhenListEmpty(t *testing.T) {
	m := newModel(nil)
	m.done = true
	next, cmd := m.Update(tickMsg{})
	got := next.(model)
	assert.True(t, got.shouldQuit())
	assert.NotNil(t, cmd)
}

func TestKeepTickingWhenDoneButLive(t *testing.T) {
	m := newModel(nil)
	m.done = true
	m.nodes = []taskgroup.Node{{ID: 1, Name: "left", State: taskgroup.Pending}}
	next, cmd := m.Update(tickMsg{})
	got := next.(model)
	assert.False(t, got.shouldQuit())
	assert.NotNil(t, cmd)
}

func TestViewResizeSwitchesLayout(t *testing.T) {
	m := newModel(nil)
	m.sync([]taskgroup.Node{
		{ID: 1, Name: "build", Pool: taskgroup.CPU, State: taskgroup.Running, Message: "part 1/4", Current: 1, Total: 4},
	})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	m = next.(model)
	narrow := strings.TrimSuffix(m.View().Content, "\n")
	assert.Contains(t, narrow, "%")
	assert.NotContains(t, narrow, "[")
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(model)
	wide := strings.TrimSuffix(m.View().Content, "\n")
	assert.Contains(t, wide, "[")
	assert.NotContains(t, wide, "%")
}
