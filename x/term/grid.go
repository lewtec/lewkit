package term

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

const scrollLimit = 2000

type glyph struct {
	r       rune
	fg      int
	bg      int
	fgMode  int
	bgMode  int
	fgIndex int
	bgIndex int
	fgRGB   uint32
	bgRGB   uint32
	bold    bool
	wide    bool
	reverse bool
}

func (g glyph) cell() Cell {
	cell := Cell{
		FG: g.fg, BG: g.bg, Bold: g.bold, Wide: g.wide, Reverse: g.reverse,
		FGMode: g.fgMode, BGMode: g.bgMode,
		FGIndex: g.fgIndex, BGIndex: g.bgIndex,
		FGRGB: g.fgRGB, BGRGB: g.bgRGB,
	}
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

	fg      int
	bg      int
	fgMode  int
	bgMode  int
	fgIndex int
	bgIndex int
	fgRGB   uint32
	bgRGB   uint32
	bold    bool
	reverse bool

	state     int
	args      [][]int
	cur       []int
	num       int
	saw       bool
	emptySlot bool
	priv      bool
	utf       []byte

	scrollTop int
	scrollBot int

	alt         bool
	savedScreen []line
	savedScroll []line
	savedR      int
	savedC      int
	savedWrap   bool
	savedRows   int
	savedCols   int

	hold  bool
	holdR int
	holdC int

	modes   uint32
	apc     []byte
	apcDrop bool
	pics    pictures
}

const (
	stateText = iota
	stateEsc
	stateCSI
	stateOSC
	stateOSCEsc
	stateAPC
	stateAPCEsc
	stateDCS
	stateDCSEsc
)

// NewGrid returns a blank screen. rows and cols below 1 become 1.
func NewGrid(rows, cols int) *Grid {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	g := &Grid{rows: rows, cols: cols, scrollBot: rows - 1}
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

// Input is the private-mode bits the process has turned on, plus the
// alternate screen when that screen is showing.
func (g *Grid) Input() uint32 {
	bits := g.modes
	if g.alt {
		bits |= InputAlt
	}
	return bits
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
	g.resetRegion()
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
		if !padGlyph(last) {
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
	case stateAPC:
		g.apcByte(b)
	case stateAPCEsc:
		g.apcEsc(b)
	case stateDCS:
		if b == 0x1b {
			g.state = stateDCSEsc
		}
	case stateDCSEsc:
		if b == '\\' {
			g.state = stateText
			return
		}
		g.state = stateEsc
		g.esc(b)
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
	row.cells[g.cursorC] = g.penGlyph(r)
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
	if g.scrollBot < g.scrollTop || g.scrollBot >= g.rows {
		g.resetRegion()
	}
	if g.cursorR == g.scrollBot && g.cursorR >= g.scrollTop {
		full := g.scrollTop == 0 && g.scrollBot == g.rows-1 && !g.alt
		g.scrollUp(1, full)
		return
	}
	if g.cursorR+1 < g.rows {
		g.cursorR++
	}
}

func (g *Grid) esc(b byte) {
	switch b {
	case '[':
		g.state = stateCSI
		g.args = g.args[:0]
		g.cur = g.cur[:0]
		g.num = 0
		g.saw = false
		g.emptySlot = false
		g.priv = false
	case ']':
		g.state = stateOSC
	case '_':
		g.state = stateAPC
		g.apc = g.apc[:0]
		g.apcDrop = false
	case 'P':
		g.state = stateDCS
	case '7':
		g.holdCursor()
		g.state = stateText
	case '8':
		g.restoreCursor()
		g.state = stateText
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
		g.num = g.num*10 + int(b-'0')
		g.saw = true
	case b == ':':
		g.pushColon()
	case b == ';':
		g.finishParam()
		g.emptySlot = true
	case b == '?':
		g.priv = true
	case b >= 0x20 && b <= 0x2f:
		// intermediate byte
	default:
		g.finishCSI()
		g.dispatch(b)
		g.state = stateText
	}
}

func (g *Grid) pushColon() {
	val := 0
	if g.saw {
		val = g.num
	}
	g.cur = append(g.cur, val)
	g.num = 0
	g.saw = false
	g.emptySlot = false
}

func (g *Grid) finishParam() {
	if g.saw || len(g.cur) > 0 {
		val := 0
		if g.saw {
			val = g.num
		}
		g.cur = append(g.cur, val)
		g.args = append(g.args, g.cur)
		g.cur = nil
	} else if g.emptySlot {
		g.args = append(g.args, []int{0})
	}
	g.num = 0
	g.saw = false
	g.emptySlot = false
}

func (g *Grid) finishCSI() {
	g.finishParam()
	if len(g.args) == 0 {
		g.args = [][]int{{0}}
	}
}

func (g *Grid) arg(i, fallback int) int {
	if i >= len(g.args) || len(g.args[i]) == 0 || g.args[i][0] == 0 {
		return fallback
	}
	return g.args[i][0]
}

func (g *Grid) dispatch(b byte) {
	if g.priv {
		g.dispatchPrivate(b)
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
	case 'L':
		g.insertLines(g.arg(0, 1))
	case 'M':
		g.deleteLines(g.arg(0, 1))
	case 'S':
		g.scrollUp(g.arg(0, 1), false)
	case 'T':
		g.scrollDown(g.arg(0, 1))
	case 'r':
		g.setRegion(g.arg(0, 1), g.arg(1, g.rows))
	case 's':
		g.holdCursor()
	case 'u':
		g.restoreCursor()
	case 'm':
		g.sgr()
	}
}

func (g *Grid) dispatchPrivate(b byte) {
	if b != 'h' && b != 'l' {
		return
	}
	on := b == 'h'
	for _, arg := range g.args {
		if len(arg) == 0 {
			continue
		}
		switch arg[0] {
		case 47, 1047, 1049:
			g.setAlt(on)
		case 1000:
			g.setMode(InputMouse, on)
		case 1002:
			g.setMode(InputMotion, on)
		case 1003:
			g.setMode(InputAnyMotion, on)
		case 1004:
			g.setMode(InputFocus, on)
		case 1006:
			g.setMode(InputSGR, on)
		case 1007:
			g.setMode(InputAltScroll, on)
		case 2004:
			g.setMode(InputPaste, on)
		}
	}
}

func (g *Grid) setMode(bit uint32, on bool) {
	if on {
		g.modes |= bit
		return
	}
	g.modes &^= bit
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
		g.screen = g.styledLines(g.rows)
		g.pics.placements = nil
	}
}

func (g *Grid) clearFrom(row, col int) {
	g.clearSpan(row, col, g.cols)
	for r := row + 1; r < g.rows; r++ {
		g.screen[r] = g.styledLine()
	}
}

func (g *Grid) clearTo(row, col int) {
	for r := range row {
		g.screen[r] = g.styledLine()
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
		g.screen[g.cursorR] = g.styledLine()
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
	erased := g.erased()
	for col := from; col < to; col++ {
		g.screen[row].cells[col] = erased
	}
	g.screen[row].wrapped = false
}

func (g *Grid) sgr() {
	if len(g.args) == 0 {
		g.resetPen()
		return
	}
	for i := 0; i < len(g.args); i++ {
		subs := g.args[i]
		if len(subs) == 0 {
			g.resetPen()
			continue
		}
		p := subs[0]
		switch {
		case p == 0:
			g.resetPen()
		case p == 1:
			g.bold = true
		case p == 7:
			g.reverse = true
		case p == 22:
			g.bold = false
		case p == 27:
			g.reverse = false
		case p >= 30 && p <= 37:
			g.setPalette(true, p-30+1)
		case p == 39:
			g.setPalette(true, 0)
		case p >= 40 && p <= 47:
			g.setPalette(false, p-40+1)
		case p == 49:
			g.setPalette(false, 0)
		case p >= 90 && p <= 97:
			g.setPalette(true, p-90+9)
		case p >= 100 && p <= 107:
			g.setPalette(false, p-100+9)
		case p == 38 || p == 48:
			fg := p == 38
			if len(subs) > 1 {
				g.applyColor(fg, subs[1:])
				continue
			}
			i += g.applyColorArgs(fg, i)
		}
	}
}

func (g *Grid) applyColor(fg bool, subs []int) {
	if len(subs) == 0 {
		return
	}
	switch subs[0] {
	case 5:
		if len(subs) >= 2 {
			g.setIndex(fg, subs[1])
		}
	case 2:
		switch {
		case len(subs) >= 5:
			g.setRGB(fg, subs[2], subs[3], subs[4])
		case len(subs) >= 4:
			g.setRGB(fg, subs[1], subs[2], subs[3])
		}
	}
}

func (g *Grid) applyColorArgs(fg bool, i int) int {
	if i+1 >= len(g.args) || len(g.args[i+1]) == 0 {
		return 0
	}
	switch g.args[i+1][0] {
	case 5:
		if i+2 >= len(g.args) || len(g.args[i+2]) == 0 {
			return 1
		}
		g.setIndex(fg, g.args[i+2][0])
		return 2
	case 2:
		if i+4 >= len(g.args) {
			return 1
		}
		g.setRGB(fg, g.args[i+2][0], g.args[i+3][0], g.args[i+4][0])
		return 4
	default:
		return 0
	}
}

func (g *Grid) resetPen() {
	g.fg, g.bg = 0, 0
	g.fgMode, g.bgMode = ColorPalette, ColorPalette
	g.fgIndex, g.bgIndex = 0, 0
	g.fgRGB, g.bgRGB = 0, 0
	g.bold = false
	g.reverse = false
}

func (g *Grid) setPalette(fg bool, n int) {
	if fg {
		g.fg = n
		g.fgMode = ColorPalette
		g.fgIndex = 0
		g.fgRGB = 0
		return
	}
	g.bg = n
	g.bgMode = ColorPalette
	g.bgIndex = 0
	g.bgRGB = 0
}

func (g *Grid) setIndex(fg bool, n int) {
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	if fg {
		g.fg = 0
		g.fgMode = ColorIndex
		g.fgIndex = n
		g.fgRGB = 0
		return
	}
	g.bg = 0
	g.bgMode = ColorIndex
	g.bgIndex = n
	g.bgRGB = 0
}

func (g *Grid) setRGB(fg bool, r, gr, b int) {
	packed := uint32(clampByte(r))<<16 | uint32(clampByte(gr))<<8 | uint32(clampByte(b))
	if fg {
		g.fg = 0
		g.fgMode = ColorRGB
		g.fgIndex = 0
		g.fgRGB = packed
		return
	}
	g.bg = 0
	g.bgMode = ColorRGB
	g.bgIndex = 0
	g.bgRGB = packed
}

func (g *Grid) reset() {
	g.alt = false
	g.savedScreen = nil
	g.savedScroll = nil
	g.hold = false
	g.modes = 0
	g.apc = g.apc[:0]
	g.apcDrop = false
	g.pics = pictures{}
	g.resetPen()
	g.screen = blankLines(g.rows, g.cols)
	g.scroll = nil
	g.cursorR, g.cursorC = 0, 0
	g.wrapPending = false
	g.resetRegion()
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

func padGlyph(gl glyph) bool {
	return gl.r == 0 && !gl.wide && gl.fg == 0 && gl.bg == 0 && !gl.bold && !gl.reverse &&
		gl.fgMode == ColorPalette && gl.bgMode == ColorPalette
}

func clampByte(n int) int {
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}

func (g *Grid) penGlyph(r rune) glyph {
	return glyph{
		r: r, fg: g.fg, bg: g.bg, bold: g.bold, reverse: g.reverse,
		fgMode: g.fgMode, bgMode: g.bgMode,
		fgIndex: g.fgIndex, bgIndex: g.bgIndex,
		fgRGB: g.fgRGB, bgRGB: g.bgRGB,
	}
}

func (g *Grid) erased() glyph {
	return glyph{
		bg: g.bg, bgMode: g.bgMode, bgIndex: g.bgIndex, bgRGB: g.bgRGB,
	}
}

func (g *Grid) styledLine() line {
	cells := blankGlyphs(g.cols)
	erased := g.erased()
	if !padGlyph(erased) {
		for i := range cells {
			cells[i] = erased
		}
	}
	return line{cells: cells}
}

func (g *Grid) styledLines(rows int) []line {
	out := make([]line, rows)
	for i := range out {
		out[i] = g.styledLine()
	}
	return out
}

func (g *Grid) resetRegion() {
	g.scrollTop = 0
	if g.rows > 0 {
		g.scrollBot = g.rows - 1
		return
	}
	g.scrollBot = 0
}

func (g *Grid) setRegion(top, bot int) {
	if top < 1 {
		top = 1
	}
	if bot > g.rows {
		bot = g.rows
	}
	if top >= bot {
		return
	}
	g.scrollTop = top - 1
	g.scrollBot = bot - 1
	g.cursorR = 0
	g.cursorC = 0
	g.wrapPending = false
}

func (g *Grid) scrollUp(n int, keep bool) {
	if n < 1 || g.scrollBot < g.scrollTop || g.scrollBot >= len(g.screen) {
		return
	}
	height := g.scrollBot - g.scrollTop + 1
	if n > height {
		n = height
	}
	top := g.scrollTop
	bot := g.scrollBot
	if keep {
		for i := range n {
			g.scroll = trimScroll(append(g.scroll, g.screen[top+i]))
		}
	}
	copy(g.screen[top:], g.screen[top+n:bot+1])
	for row := bot - n + 1; row <= bot; row++ {
		g.screen[row] = g.styledLine()
	}
}

func (g *Grid) scrollDown(n int) {
	if n < 1 || g.scrollBot < g.scrollTop || g.scrollBot >= len(g.screen) {
		return
	}
	height := g.scrollBot - g.scrollTop + 1
	if n > height {
		n = height
	}
	top := g.scrollTop
	bot := g.scrollBot
	for row := bot; row >= top+n; row-- {
		g.screen[row] = g.screen[row-n]
	}
	for row := top; row < top+n; row++ {
		g.screen[row] = g.styledLine()
	}
}

func (g *Grid) inRegion() bool {
	return g.cursorR >= g.scrollTop && g.cursorR <= g.scrollBot
}

func (g *Grid) insertLines(n int) {
	if n < 1 || !g.inRegion() || g.scrollBot >= len(g.screen) {
		return
	}
	room := g.scrollBot - g.cursorR + 1
	if n > room {
		n = room
	}
	for row := g.scrollBot; row >= g.cursorR+n; row-- {
		g.screen[row] = g.screen[row-n]
	}
	for row := g.cursorR; row < g.cursorR+n; row++ {
		g.screen[row] = g.styledLine()
	}
}

func (g *Grid) deleteLines(n int) {
	if n < 1 || !g.inRegion() || g.scrollBot >= len(g.screen) {
		return
	}
	room := g.scrollBot - g.cursorR + 1
	if n > room {
		n = room
	}
	copy(g.screen[g.cursorR:], g.screen[g.cursorR+n:g.scrollBot+1])
	for row := g.scrollBot - n + 1; row <= g.scrollBot; row++ {
		g.screen[row] = g.styledLine()
	}
}

func (g *Grid) holdCursor() {
	g.hold = true
	g.holdR = g.cursorR
	g.holdC = g.cursorC
}

func (g *Grid) restoreCursor() {
	if !g.hold {
		return
	}
	g.cursorR = g.holdR
	g.cursorC = g.holdC
	g.wrapPending = false
	g.clampCursor()
}

func (g *Grid) setAlt(on bool) {
	if on == g.alt {
		return
	}
	if on {
		g.savedScreen = cloneLines(g.screen)
		g.savedScroll = cloneLines(g.scroll)
		g.savedR, g.savedC = g.cursorR, g.cursorC
		g.savedWrap = g.wrapPending
		g.savedRows, g.savedCols = g.rows, g.cols
		g.screen = g.styledLines(g.rows)
		g.scroll = nil
		g.cursorR, g.cursorC = 0, 0
		g.wrapPending = false
		g.alt = true
		g.resetRegion()
		return
	}
	mainScreen := g.savedScreen
	mainScroll := g.savedScroll
	mainR, mainC := g.savedR, g.savedC
	mainWrap := g.savedWrap
	mainRows, mainCols := g.savedRows, g.savedCols
	g.savedScreen = nil
	g.savedScroll = nil
	g.alt = false
	curRows, curCols := g.rows, g.cols
	if mainScreen == nil {
		g.screen = g.styledLines(curRows)
		g.scroll = nil
		g.cursorR, g.cursorC = 0, 0
		g.wrapPending = false
		g.resetRegion()
		return
	}
	g.screen = mainScreen
	g.scroll = mainScroll
	g.cursorR, g.cursorC = mainR, mainC
	g.wrapPending = mainWrap
	g.rows, g.cols = mainRows, mainCols
	if mainRows != curRows || mainCols != curCols {
		g.Reflow(curRows, curCols)
		return
	}
	g.resetRegion()
}

func cloneLines(rows []line) []line {
	out := make([]line, len(rows))
	for i, row := range rows {
		out[i] = line{cells: append([]glyph(nil), row.cells...), wrapped: row.wrapped}
	}
	return out
}
