package progress

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func TestFormatBarLineWide(t *testing.T) {
	line := formatBarLine(barEntry{
		title:    "bundle.tar.gz",
		subtitle: "5.0 MiB / 10.0 MiB",
		pool:     taskgroup.Internet,
		percent:  0.5,
	}, 80)
	if !strings.HasPrefix(line, "🌐 bundle.tar.gz: 5.0 MiB / 10.0 MiB [") {
		t.Fatalf("wide = %q, want emoji message [bar]", line)
	}
	if !strings.HasSuffix(line, "]") {
		t.Fatalf("wide = %q, want trailing ]", line)
	}
	if cellWidth(line) != 80 {
		t.Fatalf("wide width = %d, want 80 (%q)", cellWidth(line), line)
	}
}

func TestFormatBarLineNarrow(t *testing.T) {
	line := formatBarLine(barEntry{
		title:    "bundle.tar.gz",
		subtitle: "5.0 MiB / 10.0 MiB",
		pool:     taskgroup.Internet,
		percent:  0.5,
	}, 40)
	if !strings.HasPrefix(line, "🌐  50.0% ") {
		t.Fatalf("narrow = %q, want emoji percent message", line)
	}
	if strings.Contains(line, "[") {
		t.Fatalf("narrow kept a bar: %q", line)
	}
	if cellWidth(line) > 40 {
		t.Fatalf("narrow width = %d > 40 (%q)", cellWidth(line), line)
	}
}

func TestFormatBarLinePools(t *testing.T) {
	tests := []struct {
		pool  taskgroup.PoolKind
		emoji string
	}{
		{taskgroup.Control, "🔧"},
		{taskgroup.IO, "💾"},
		{taskgroup.CPU, "🧠"},
		{taskgroup.Internet, "🌐"},
	}
	for _, tt := range tests {
		line := formatBarLine(barEntry{title: "t", subtitle: "s", pool: tt.pool, percent: 0}, 80)
		if !strings.HasPrefix(line, tt.emoji+" ") {
			t.Errorf("pool %v: line = %q, want emoji %s", tt.pool, line, tt.emoji)
		}
	}
}

func TestSyncKeysByID(t *testing.T) {
	m := newModel(nil)
	m.sync([]taskgroup.Node{
		{ID: 1, Name: "fetch", Pool: taskgroup.Internet, State: taskgroup.Running, Message: "a", Current: 1, Total: 2},
		{ID: 2, Name: "fetch", Pool: taskgroup.Internet, State: taskgroup.Running, Message: "b", Current: 2, Total: 2},
	})
	if len(m.bars) != 2 {
		t.Fatalf("bars = %d, want 2", len(m.bars))
	}
	if m.bars["1"].subtitle != "a" || m.bars["2"].subtitle != "b" {
		t.Fatalf("bars = %+v", m.bars)
	}
	m.sync([]taskgroup.Node{
		{ID: 1, Name: "fetch", Pool: taskgroup.Internet, State: taskgroup.Running, Message: "done-ish", Current: 2, Total: 2},
		{ID: 2, Name: "fetch", Pool: taskgroup.Internet, State: taskgroup.Done, Message: "b", Current: 2, Total: 2},
	})
	if _, ok := m.bars["2"]; ok {
		t.Fatal("finished id 2 should drop")
	}
	if m.bars["1"].percent != 1 {
		t.Errorf("id 1 percent = %v, want 1", m.bars["1"].percent)
	}
}

func TestSyncSkipsIndeterminate(t *testing.T) {
	m := newModel(nil)
	m.sync([]taskgroup.Node{
		{ID: 1, Name: "wait", Pool: taskgroup.Control, State: taskgroup.Running, Total: -1},
		{ID: 2, Name: "work", Pool: taskgroup.CPU, State: taskgroup.Running, Message: "go", Current: 0, Total: 1},
	})
	if len(m.bars) != 1 {
		t.Fatalf("bars = %d, want 1", len(m.bars))
	}
	if m.bars["2"].subtitle != "go" {
		t.Errorf("subtitle = %q, want go", m.bars["2"].subtitle)
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
