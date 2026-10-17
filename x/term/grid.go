package term

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

const scrollLimit = 2000

type glyph struct {
	r    rune
	fg   int
	bg   int
	bold bool
	wide bool
}

func (g glyph) cell() Cell {
	cell := Cell{FG: g.fg, BG: g.bg, Bold: g.bold, Wide: g.wide}
	if g.r != 0 {
		cell.Rune = string(g.r)
	}
	return cell
}

type line struct {
	cells   []glyph
	wrapped bool
}

// Grid is the pane's visible screen and the scrollback above it.
// Feed accepts terminal bytes. Reflow rewraps logical lines to a new size.
type Grid struct {
	rows int
	cols int

	screen []line
	scroll []line

	cursorR     int
	cursorC     int
	wrapPending bool

	fg   int
	bg   int
	bold bool

	state  int
	params []int
	param  int
	saw    bool
	priv   bool
	utf    []byte
}

const (
	stateText = iota
	stateEsc
	stateCSI
	stateOSC
	stateOSCEsc
)

// NewGrid returns a blank screen. rows and cols below 1 become 1.
func NewGrid(rows, cols int) *Grid {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	g := &Grid{rows: rows, cols: cols}
	g.screen = blankLines(rows, cols)
	return g
}

// Size is the screen the process was last sized to.
func (g *Grid) Size() (rows, cols int) {
	return g.rows, g.cols
}

// Cursor is the cursor column clamped into the screen.
func (g *Grid) Cursor() (row, col int) {
	col = g.cursorC
	if col >= g.cols {
		col = g.cols - 1
	}
	if col < 0 {
		col = 0
	}
	row = g.cursorR
	if row >= g.rows {
		row = g.rows - 1
	}
	if row < 0 {
		row = 0
	}
	return row, col
}

// Snapshot copies the visible cells in row-major order.
func (g *Grid) Snapshot() (rows, cols, cursorRow, cursorCol int, cells []Cell) {
	rows, cols = g.rows, g.cols
	cursorRow, cursorCol = g.Cursor()
	cells = make([]Cell, 0, rows*cols)
	for _, row := range g.screen {
		for _, item := range row.cells {
			cells = append(cells, item.cell())
		}
	}
	return rows, cols, cursorRow, cursorCol, cells
}

// Text is the visible screen, one line per row, with trailing blanks removed.
func (g *Grid) Text() string {
	rows, cols, _, _, cells := g.Snapshot()
	pane := Pane{PaneInfo: PaneInfo{Rows: rows, Cols: cols}, Cells: cells}
	parts := make([]string, rows)
	for row := range rows {
		parts[row] = pane.Line(row)
	}
	// Keep interior blank rows so a cursor row stays addressable.
	end := len(parts)
	for end > 0 && parts[end-1] == "" {
		end--
	}
	if end == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:end] {
		out += "\n" + part
	}
	return out
}

// Feed applies terminal bytes to the grid.
func (g *Grid) Feed(p []byte) {
	for _, b := range p {
		g.byte(b)
	}
}

// Reflow rewraps scrollback and the screen to rows and cols.
// The cursor stays on the same logical cell.
func (g *Grid) Reflow(rows, cols int) {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	lines, lineIndex, slot := g.logical()
	phys := make([]line, 0, len(lines))
	cursorPhys := 0
	cursorCol := 0
	for i, item := range lines {
		parts := rechunk(item, cols)
		if i == lineIndex {
			left := slot
			placed := false
			for pi := range parts {
				if left < cols {
					cursorPhys = len(phys) + pi
					cursorCol = left
					placed = true
					break
				}
				left -= cols
			}
			if !placed {
				cursorPhys = len(phys) + len(parts) - 1
				cursorCol = cols
			}
		}
		phys = append(phys, parts...)
	}
	if len(phys) == 0 {
		phys = blankLines(1, cols)
	}
	start, end := window(len(phys), cursorPhys, rows)
	for len(phys) < end {
		phys = append(phys, blankLine(cols))
	}
	g.scroll = trimScroll(append([]line(nil), phys[:start]...))
	g.screen = append([]line(nil), phys[start:end]...)
	g.rows = rows
	g.cols = cols
	g.cursorR = cursorPhys - start
	g.cursorC = cursorCol
	if g.cursorC > cols {
		g.cursorC = cols
	}
	g.wrapPending = false
}

func (g *Grid) logical() (lines [][]glyph, lineIndex, slot int) {
	all := append(append([]line{}, g.scroll...), g.screen...)
	phys := len(g.scroll) + g.cursorR
	if phys < 0 {
		phys = 0
	}
	if phys > len(all) {
		phys = len(all)
	}
	var cur []glyph
	flush := func() {
		lines = append(lines, cur)
		cur = nil
	}
	for i, row := range all {
		if i == phys {
			lineIndex = len(lines)
			slot = len(cur) + g.cursorC
		}
		cells := row.cells
		if !row.wrapped {
			cells = trimPad(cells)
			// Rows under the cursor that were never written are the viewport,
			// not blank lines in the scrollback.
			if len(cells) == 0 && i > phys {
				continue
			}
			cur = append(cur, cells...)
			flush()
			continue
		}
		cur = append(cur, cells...)
	}
	if cur != nil {
		flush()
	}
	if len(lines) == 0 {
		lines = [][]glyph{{}}
	}
	if phys >= len(all) {
		lineIndex = len(lines) - 1
		slot = len(lines[lineIndex])
	}
	if lineIndex >= len(lines) {
		lineIndex = len(lines) - 1
	}
	if slot < 0 {
		slot = 0
	}
	return lines, lineIndex, slot
}

func window(n, cursor, rows int) (start, end int) {
	if n <= rows {
		return 0, rows
	}
	below := n - cursor - 1
	if below < 0 {
		below = 0
	}
	if below > rows-1 {
		below = rows - 1
	}
	end = cursor + below + 1
	start = end - rows
	if start < 0 {
		start = 0
		end = rows
	}
	if end > n {
		end = n
		start = end - rows
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func rechunk(cells []glyph, cols int) []line {
	if cols < 1 {
		cols = 1
	}
	if len(cells) == 0 {
		return blankLines(1, cols)
	}
	var rows []line
	cur := blankGlyphs(cols)
	col := 0
	flush := func(wrapped bool) {
		rows = append(rows, line{cells: cur, wrapped: wrapped})
		cur = blankGlyphs(cols)
		col = 0
	}
	for i := 0; i < len(cells); i++ {
		item := cells[i]
		if item.wide {
			continue
		}
		w := glyphWidth(item.r)
		if col+w > cols {
			flush(true)
		}
		cur[col] = item
		if w == 2 && col+1 < cols {
			cur[col+1] = glyph{wide: true}
		}
		col += w
	}
	flush(false)
	return rows
}

func glyphWidth(r rune) int {
	if r == 0 {
		return 1
	}
	w := runewidth.RuneWidth(r)
	if w >= 2 {
		return 2
	}
	return 1
}

func trimPad(cells []glyph) []glyph {
	end := len(cells)
	for end > 0 {
		last := cells[end-1]
		if last.wide {
			end--
			continue
		}
		if last.r != 0 || last.fg != 0 || last.bg != 0 || last.bold {
			break
		}
		end--
	}
	return cells[:end]
}

func trimScroll(rows []line) []line {
	if len(rows) <= scrollLimit {
		return rows
	}
	return rows[len(rows)-scrollLimit:]
}

func (g *Grid) byte(b byte) {
	switch g.state {
	case stateText:
		g.text(b)
	case stateEsc:
		g.esc(b)
	case stateCSI:
		g.csi(b)
	case stateOSC:
		if b == 0x07 {
			g.state = stateText
			return
		}
		if b == 0x1b {
			g.state = stateOSCEsc
		}
	case stateOSCEsc:
		g.state = stateText
	}
}

func (g *Grid) text(b byte) {
	switch b {
	case 0x1b:
		g.flushUTF()
		g.state = stateEsc
		return
	case '\n', '\v', '\f':
		g.flushUTF()
		g.wrapPending = false
		g.cursorC = 0
		g.down()
		return
	case '\r':
		g.flushUTF()
		g.wrapPending = false
		g.cursorC = 0
		return
	case '\b', 0x7f:
		g.flushUTF()
		g.wrapPending = false
		if g.cursorC > 0 {
			g.cursorC--
		}
		return
	case '\t':
		g.flushUTF()
		g.tab()
		return
	default:
		if b < 0x20 {
			return
		}
	}
	g.utf = append(g.utf, b)
	if utf8.FullRune(g.utf) {
		g.flushUTF()
	}
}

func (g *Grid) flushUTF() {
	if len(g.utf) == 0 {
		return
	}
	r, _ := utf8.DecodeRune(g.utf)
	g.utf = g.utf[:0]
	if r == utf8.RuneError {
		r = '\uFFFD'
	}
	g.put(r)
}

func (g *Grid) tab() {
	stop := (g.cursorC/8 + 1) * 8
	if stop > g.cols {
		stop = g.cols
	}
	g.cursorC = stop
	if g.cursorC >= g.cols {
		g.wrapPending = true
	}
}

func (g *Grid) put(r rune) {
	if runewidth.RuneWidth(r) == 0 && r != 0 {
		return
	}
	if g.wrapPending {
		g.softWrap()
	}
	w := glyphWidth(r)
	if g.cursorC+w > g.cols {
		g.softWrap()
	}
	row := &g.screen[g.cursorR]
	row.cells[g.cursorC] = glyph{r: r, fg: g.fg, bg: g.bg, bold: g.bold}
	if w == 2 {
		row.cells[g.cursorC+1] = glyph{wide: true}
	}
	g.cursorC += w
	if g.cursorC >= g.cols {
		g.wrapPending = true
	}
}

func (g *Grid) softWrap() {
	g.screen[g.cursorR].wrapped = true
	g.wrapPending = false
	g.cursorC = 0
	g.down()
}

func (g *Grid) down() {
	if g.cursorR+1 < g.rows {
		g.cursorR++
		return
	}
	g.scroll = trimScroll(append(g.scroll, g.screen[0]))
	copy(g.screen, g.screen[1:])
	g.screen[g.rows-1] = blankLine(g.cols)
}

func (g *Grid) esc(b byte) {
	switch b {
	case '[':
		g.state = stateCSI
		g.params = g.params[:0]
		g.param = 0
		g.saw = false
		g.priv = false
	case ']':
		g.state = stateOSC
	case 'c':
		g.reset()
		g.state = stateText
	default:
		g.state = stateText
	}
}

func (g *Grid) csi(b byte) {
	switch {
	case b >= '0' && b <= '9':
		g.param = g.param*10 + int(b-'0')
		g.saw = true
	case b == ';':
		g.pushParam()
	case b == '?':
		g.priv = true
	case b >= 0x20 && b <= 0x2f:
		// intermediate byte
	default:
		g.pushParam()
		g.dispatch(b)
		g.state = stateText
	}
}

func (g *Grid) pushParam() {
	if !g.saw {
		g.params = append(g.params, 0)
	} else {
		g.params = append(g.params, g.param)
	}
	g.param = 0
	g.saw = false
}

func (g *Grid) arg(i, fallback int) int {
	if i >= len(g.params) || g.params[i] == 0 {
		return fallback
	}
	return g.params[i]
}

func (g *Grid) dispatch(b byte) {
	if g.priv {
		return
	}
	switch b {
	case 'A':
		g.moveR(-g.arg(0, 1))
	case 'B':
		g.moveR(g.arg(0, 1))
	case 'C':
		g.moveC(g.arg(0, 1))
	case 'D':
		g.moveC(-g.arg(0, 1))
	case 'G':
		g.cursorC = g.arg(0, 1) - 1
		g.clampCursor()
	case 'd':
		g.cursorR = g.arg(0, 1) - 1
		g.clampCursor()
	case 'H', 'f':
		g.cursorR = g.arg(0, 1) - 1
		g.cursorC = g.arg(1, 1) - 1
		g.clampCursor()
		g.wrapPending = false
	case 'J':
		g.eraseDisplay(g.arg(0, 0))
	case 'K':
		g.eraseLine(g.arg(0, 0))
	case 'm':
		g.sgr()
	}
}

func (g *Grid) moveR(d int) {
	g.wrapPending = false
	g.cursorR += d
	g.clampCursor()
}

func (g *Grid) moveC(d int) {
	g.wrapPending = false
	g.cursorC += d
	g.clampCursor()
}

func (g *Grid) clampCursor() {
	if g.cursorR < 0 {
		g.cursorR = 0
	}
	if g.cursorR >= g.rows {
		g.cursorR = g.rows - 1
	}
	if g.cursorC < 0 {
		g.cursorC = 0
	}
	if g.cursorC >= g.cols {
		g.cursorC = g.cols - 1
	}
}

func (g *Grid) eraseDisplay(mode int) {
	switch mode {
	case 0:
		g.clearFrom(g.cursorR, g.cursorC)
	case 1:
		g.clearTo(g.cursorR, g.cursorC)
	default:
		g.screen = blankLines(g.rows, g.cols)
	}
}

func (g *Grid) clearFrom(row, col int) {
	g.clearSpan(row, col, g.cols)
	for r := row + 1; r < g.rows; r++ {
		g.screen[r] = blankLine(g.cols)
	}
}

func (g *Grid) clearTo(row, col int) {
	for r := range row {
		g.screen[r] = blankLine(g.cols)
	}
	g.clearSpan(row, 0, col+1)
}

func (g *Grid) eraseLine(mode int) {
	switch mode {
	case 0:
		g.clearSpan(g.cursorR, g.cursorC, g.cols)
	case 1:
		g.clearSpan(g.cursorR, 0, g.cursorC+1)
	default:
		g.screen[g.cursorR] = blankLine(g.cols)
	}
}

func (g *Grid) clearSpan(row, from, to int) {
	if row < 0 || row >= len(g.screen) {
		return
	}
	if from < 0 {
		from = 0
	}
	if to > g.cols {
		to = g.cols
	}
	for col := from; col < to; col++ {
		g.screen[row].cells[col] = glyph{}
	}
	g.screen[row].wrapped = false
}

func (g *Grid) sgr() {
	if len(g.params) == 0 {
		g.fg, g.bg, g.bold = 0, 0, false
		return
	}
	for _, p := range g.params {
		switch {
		case p == 0:
			g.fg, g.bg, g.bold = 0, 0, false
		case p == 1:
			g.bold = true
		case p == 22:
			g.bold = false
		case p >= 30 && p <= 37:
			g.fg = p - 30 + 1
		case p == 39:
			g.fg = 0
		case p >= 40 && p <= 47:
			g.bg = p - 40 + 1
		case p == 49:
			g.bg = 0
		case p >= 90 && p <= 97:
			g.fg = p - 90 + 9
		case p >= 100 && p <= 107:
			g.bg = p - 100 + 9
		}
	}
}

func (g *Grid) reset() {
	g.screen = blankLines(g.rows, g.cols)
	g.scroll = nil
	g.cursorR, g.cursorC = 0, 0
	g.wrapPending = false
	g.fg, g.bg, g.bold = 0, 0, false
	g.utf = g.utf[:0]
}

func blankLines(rows, cols int) []line {
	out := make([]line, rows)
	for i := range out {
		out[i] = blankLine(cols)
	}
	return out
}

func blankLine(cols int) line {
	return line{cells: blankGlyphs(cols)}
}

func blankGlyphs(cols int) []glyph {
	return make([]glyph, cols)
}
