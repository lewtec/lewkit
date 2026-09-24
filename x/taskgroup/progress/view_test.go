package progress

import (
	"context"
	"strings"
	"sync/atomic"
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
	assert.Equal(t, "🔧", rows[0].emoji)
	assert.Equal(t, "🔧", rows[3].emoji)
	assert.Equal(t, "🔧", rows[4].emoji)
}

func TestLayoutTaskEmojiOverride(t *testing.T) {
	rows := layout([]taskgroup.Node{{
		ID: 1, Name: "tmp", Pool: taskgroup.IO, Emoji: "📁", State: taskgroup.Done,
	}})
	require.Len(t, rows, 1)
	assert.Equal(t, "📁", rows[0].emoji)
	line := formatRow(rows[0], 80)
	assert.True(t, strings.HasPrefix(line, "✔ 📁 tmp"), line)
	assert.NotContains(t, line, "💾")
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
		tree:  "├ ",
		emoji: "🧠",
	}, 80)
	assert.True(t, strings.HasPrefix(line, "├ ▶ 🧠 build: part 1/4"), line)
	assert.True(t, strings.HasSuffix(line, plainBar(0.25, barWidth)), line)
	assert.Equal(t, barWidth, barCells(line), line)
	assert.Equal(t, 80, cellWidth(line), line)
	assert.NotContains(t, line, "[")
	assert.Contains(t, line, barFill)
	assert.Contains(t, line, barEmpty)
}

func TestFormatRowBarsAlign(t *testing.T) {
	short := formatRow(treeRow{
		node:  taskgroup.Node{Name: "a", Pool: taskgroup.CPU, State: taskgroup.Running, Current: 1, Total: 4},
		emoji: "🧠",
	}, 80)
	long := formatRow(treeRow{
		node:  taskgroup.Node{Name: "compile-frontend", Pool: taskgroup.CPU, State: taskgroup.Running, Current: 1, Total: 4},
		emoji: "🧠",
	}, 80)
	bar := plainBar(0.25, barWidth)
	assert.True(t, strings.HasSuffix(short, bar), short)
	assert.True(t, strings.HasSuffix(long, bar), long)
	assert.Equal(t, barWidth, barCells(short))
	assert.Equal(t, barWidth, barCells(long))
	assert.Equal(t, 80, cellWidth(short))
	assert.Equal(t, 80, cellWidth(long))
}

func TestFormatRowUsesFullWidth(t *testing.T) {
	withBar := treeRow{
		node:  taskgroup.Node{Name: "build", Pool: taskgroup.CPU, State: taskgroup.Running, Current: 1, Total: 4},
		emoji: "🧠",
	}
	noBar := treeRow{
		node:  taskgroup.Node{Name: "install", Pool: taskgroup.IO, State: taskgroup.Pending},
		emoji: "💾",
	}
	for _, width := range []int{80, 160} {
		barLine := formatRow(withBar, width)
		plain := formatRow(noBar, width)
		assert.Equal(t, width, cellWidth(barLine), barLine)
		assert.Equal(t, width, cellWidth(plain), plain)
		assert.Equal(t, barWidth, barCells(barLine))
	}
}

func barCells(line string) int {
	return strings.Count(line, barFill) + strings.Count(line, barEmpty)
}

func TestPlainBarFill(t *testing.T) {
	assert.Equal(t, strings.Repeat(barEmpty, 10), plainBar(0, 10))
	assert.Equal(t, strings.Repeat(barFill, 5)+strings.Repeat(barEmpty, 5), plainBar(0.5, 10))
	assert.Equal(t, strings.Repeat(barFill, 10), plainBar(1, 10))
}

func TestFormatRowPendingNoBar(t *testing.T) {
	line := formatRow(treeRow{
		node:  taskgroup.Node{Name: "install", Pool: taskgroup.IO, State: taskgroup.Pending},
		tree:  "└ ",
		emoji: "💾",
	}, 80)
	assert.True(t, strings.HasPrefix(line, "└ ⏸ 💾 install"), line)
	assert.NotContains(t, line, barFill)
	assert.NotContains(t, line, "[")
	assert.Equal(t, 80, cellWidth(line), line)
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
	assert.NotContains(t, narrow, barFill)
	next, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(model)
	wide := strings.TrimSuffix(m.View().Content, "\n")
	assert.Contains(t, wide, barFill)
	assert.NotContains(t, wide, "%")
}

func TestRunWithoutGoSkipsProgram(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_TUI", "1")
	s, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	require.NoError(t, Run(s, ctx, func(context.Context) error { return nil }))
}

func TestRunNonInteractiveWaits(t *testing.T) {
	t.Setenv("TERM", "dumb")
	t.Setenv("LEWKIT_FORCE_TUI", "")
	s, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	var ran atomic.Bool
	require.NoError(t, Run(s, ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, "t", taskgroup.CPU, func(context.Context, *taskgroup.Status) error {
			ran.Store(true)
			return nil
		})
		return nil
	}))
	assert.True(t, ran.Load())
}
