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
	barWidth         = 20
	barFill          = "━"
	barEmpty         = "─"
)

type model struct {
	session *taskgroup.Session
	nodes   []taskgroup.Node
	live    []string
	width   int
	max     int
	done    bool
}

func (m *model) requestStop() {
	if m.session != nil {
		m.session.Cancel(context.Canceled)
	}
}

type treeRow struct {
	node   taskgroup.Node
	tree   string
	emoji  string
	hidden int
}

func newModel(s *taskgroup.Session) model {
	return model{
		session: s,
		width:   defaultTermWidth,
		max:     defaultMaxRows,
	}
}

func (m *model) sync(nodes []taskgroup.Node) {
	m.nodes = nodes
}

func (m model) shouldQuit() bool {
	return m.done && len(m.nodes) == 0 && len(m.live) == 0
}

func (m model) View() (view tea.View) {
	view.KeyboardEnhancements = tea.KeyboardEnhancements{}
	view.AltScreen = false
	view.MouseMode = tea.MouseModeNone
	if len(m.nodes) == 0 && len(m.live) == 0 {
		view.SetContent("")
		return
	}
	width := m.width
	if width <= 0 {
		width = defaultTermWidth
	}
	var buf bytes.Buffer
	for _, row := range m.live {
		buf.WriteString(padCells(row, width))
		buf.WriteByte('\n')
	}
	writeRows(&buf, m.nodes, width)
	view.SetContent(buf.String())
	return
}

func layout(nodes []taskgroup.Node) []treeRow {
	if len(nodes) == 0 {
		return nil
	}
	idx := make(map[taskgroup.ID]int, len(nodes))
	for i, n := range nodes {
		idx[n.ID] = i
	}
	depth := make([]int, len(nodes))
	shownKids := make([]int, len(nodes))
	lastAt := make(map[taskgroup.ID]int, len(nodes))
	for i, n := range nodes {
		if p, ok := idx[n.Parent]; ok {
			depth[i] = depth[p] + 1
			shownKids[p]++
		}
		lastAt[n.Parent] = i
	}
	out := make([]treeRow, len(nodes))
	last := make([]bool, 0, 8)
	for i, n := range nodes {
		d := depth[i]
		isLast := lastAt[n.Parent] == i
		var b strings.Builder
		for k := 0; k < d; k++ {
			if k == d-1 {
				if isLast {
					b.WriteString("└ ")
				} else {
					b.WriteString("├ ")
				}
				continue
			}
			if last[k] {
				b.WriteString("  ")
			} else {
				b.WriteString("│ ")
			}
		}
		if d+1 > len(last) {
			last = append(last, make([]bool, d+1-len(last))...)
		}
		last = last[:d+1]
		last[d] = isLast
		hidden := n.LiveChildren - shownKids[i]
		if hidden < 0 {
			hidden = 0
		}
		out[i] = treeRow{node: n, tree: b.String(), emoji: n.TaskEmoji(), hidden: hidden}
	}
	return out
}

func formatRow(r treeRow, width int) string {
	if width <= 0 {
		width = defaultTermWidth
	}
	n := r.node
	prefix := r.tree + stateGlyph(n.State) + " " + r.emoji + " "
	msg := rowMessage(n)
	if r.hidden > 0 {
		msg += " +" + strconv.Itoa(r.hidden)
	}
	rest := width - cellWidth(prefix)
	var line string
	switch {
	case rest < 1:
		line = clipCells(prefix, width)
	case n.Total <= 0 || percent(n) < 0:
		line = prefix + clipCells(msg, rest)
	case width < narrowTermWidth:
		line = prefix + formatNarrowBody(msg, percent(n), rest)
	default:
		line = prefix + formatWideBody(msg, percent(n), rest)
	}
	return padCells(line, width)
}

func rowMessage(n taskgroup.Node) string {
	title := n.Name
	if title == "" {
		title = strconv.FormatUint(uint64(n.ID), 10)
	}
	sub := n.Message
	if sub == "" && n.State == taskgroup.Running {
		sub = "running"
	}
	switch {
	case title != "" && sub != "":
		return title + ": " + sub
	case title != "":
		return title
	default:
		return sub
	}
}

func percent(n taskgroup.Node) float64 {
	if n.Total <= 0 {
		return -1
	}
	return float64(n.Current) / float64(n.Total)
}

func stateGlyph(st taskgroup.State) string {
	switch st {
	case taskgroup.Pending:
		return "⏸"
	case taskgroup.Running:
		return "▶"
	case taskgroup.Done:
		return "✔"
	case taskgroup.Failed:
		return "⚠"
	default:
		return "•"
	}
}

func formatNarrowBody(msg string, pct float64, width int) string {
	head := formatPercent(pct) + " "
	rest := width - cellWidth(head)
	if rest < 0 {
		return clipCells(head, width)
	}
	return head + clipCells(msg, rest)
}

func formatWideBody(msg string, pct float64, width int) string {
	const gap = 1
	inner := barWidth
	if width < gap+minBarInner+1 {
		return formatNarrowBody(msg, pct, width)
	}
	if gap+inner+1 > width {
		inner = width - gap - 1
	}
	if inner < minBarInner {
		return formatNarrowBody(msg, pct, width)
	}
	return padCells(msg, width-gap-inner) + " " + plainBar(pct, inner)
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

func plainBar(pct float64, width int) string {
	if width <= 0 {
		width = barWidth
	}
	pct = min(max(pct, 0), 1)
	filled := min(int(pct*float64(width)+0.5), width)
	return strings.Repeat(barFill, filled) + strings.Repeat(barEmpty, width-filled)
}

func padCells(s string, width int) string {
	s = clipCells(s, width)
	pad := width - cellWidth(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
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
