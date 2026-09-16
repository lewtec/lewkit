package progress

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/mattn/go-runewidth"
)

const (
	defaultTermWidth = 80
	narrowTermWidth  = 56
	minBarInner      = 8
)

type barEntry struct {
	title    string
	subtitle string
	pool     taskgroup.PoolKind
	percent  float64
}

type model struct {
	session *taskgroup.Session
	cancel  context.CancelFunc
	bars    map[string]barEntry
	order   []string
	width   int
	max     int
}

func newModel(s *taskgroup.Session) model {
	return model{
		session: s,
		bars:    make(map[string]barEntry),
		width:   defaultTermWidth,
		max:     defaultMaxRows,
	}
}

func (m *model) sync(nodes []taskgroup.Node) {
	seen := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		if n.State != taskgroup.Running || n.Total <= 0 {
			continue
		}
		id := strconv.FormatUint(uint64(n.ID), 10)
		seen[id] = struct{}{}
		pct := float64(n.Current) / float64(n.Total)
		subtitle := n.Message
		if subtitle == "" {
			subtitle = "running"
		}
		title := n.Name
		if title == "" {
			title = id
		}
		if prev, ok := m.bars[id]; ok {
			prev.subtitle = subtitle
			prev.percent = pct
			m.bars[id] = prev
			continue
		}
		m.bars[id] = barEntry{
			title:    title,
			subtitle: subtitle,
			pool:     n.Pool,
			percent:  pct,
		}
		m.order = append(m.order, id)
	}
	for id := range m.bars {
		if _, ok := seen[id]; !ok {
			delete(m.bars, id)
		}
	}
	if len(m.order) > len(m.bars)+8 {
		out := m.order[:0]
		for _, id := range m.order {
			if _, ok := m.bars[id]; ok {
				out = append(out, id)
			}
		}
		m.order = out
	}
}

func (m model) View() (view tea.View) {
	view.KeyboardEnhancements = tea.KeyboardEnhancements{}
	view.AltScreen = false
	view.MouseMode = tea.MouseModeNone
	if len(m.bars) == 0 {
		view.SetContent("")
		return
	}
	width := m.width
	if width <= 0 {
		width = defaultTermWidth
	}
	var buf bytes.Buffer
	for _, id := range m.order {
		b, ok := m.bars[id]
		if !ok {
			continue
		}
		buf.WriteString(formatBarLine(b, width))
		buf.WriteByte('\n')
	}
	view.SetContent(buf.String())
	return
}

func formatBarLine(b barEntry, width int) string {
	if width <= 0 {
		width = defaultTermWidth
	}
	emoji := poolEmoji(b.pool)
	msg := barMessage(b)
	if width < narrowTermWidth {
		return formatNarrowBar(emoji, msg, b.percent, width)
	}
	return formatWideBar(emoji, msg, b.percent, width)
}

func barMessage(b barEntry) string {
	switch {
	case b.title != "" && b.subtitle != "":
		return b.title + ": " + b.subtitle
	case b.title != "":
		return b.title
	default:
		return b.subtitle
	}
}

func formatNarrowBar(emoji, msg string, pct float64, width int) string {
	prefix := emoji + " " + formatPercent(pct) + " "
	rest := width - cellWidth(prefix)
	if rest < 0 {
		return clipCells(prefix, width)
	}
	return prefix + clipCells(msg, rest)
}

func formatWideBar(emoji, msg string, pct float64, width int) string {
	fixed := cellWidth(emoji) + 1 + 1 + 2
	inner := width - fixed - cellWidth(msg)
	if inner < minBarInner {
		msg = clipCells(msg, width-fixed-minBarInner)
		inner = width - fixed - cellWidth(msg)
	}
	if inner < 1 {
		return formatNarrowBar(emoji, msg, pct, width)
	}
	return emoji + " " + msg + " " + plainBar(pct, inner)
}

func formatPercent(pct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	return fmt.Sprintf("%5.1f%%", pct*100)
}

func poolEmoji(p taskgroup.PoolKind) string {
	switch p {
	case taskgroup.Control:
		return "🔧"
	case taskgroup.IO:
		return "💾"
	case taskgroup.CPU:
		return "🧠"
	case taskgroup.Internet:
		return "🌐"
	default:
		return "•"
	}
}

func plainBar(pct float64, width int) string {
	if width <= 0 {
		width = 30
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	filled := min(int(pct*float64(width)+0.5), width)
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
}

func cellWidth(s string) int {
	return runewidth.StringWidth(s)
}

func clipCells(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if cellWidth(s) <= max {
		return s
	}
	return runewidth.Truncate(s, max, "")
}
