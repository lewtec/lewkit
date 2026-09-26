package report

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/term"
	"github.com/lewtec/lewkit/x/path"
)

// WriteRustc writes rustc-style diagnostics: header, snippet, underline, notes.
// Color is on for a terminal and off for a pipe, a buffer, or NO_COLOR.
func WriteRustc(w io.Writer, root string, findings []Finding) error {
	return newRustcWriter(w, root, autoColor(w)).write(findings)
}

func autoColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

type rustcWriter struct {
	w     io.Writer
	root  string
	color bool
}

func newRustcWriter(w io.Writer, root string, color bool) *rustcWriter {
	return &rustcWriter{w: w, root: root, color: color}
}

func (rw *rustcWriter) write(findings []Finding) error {
	for i, f := range findings {
		if i > 0 {
			if _, err := io.WriteString(rw.w, "\n"); err != nil {
				return err
			}
		}
		if err := rw.finding(f); err != nil {
			return err
		}
	}
	return nil
}

func (rw *rustcWriter) paint(s, code string) string {
	if !rw.color || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (rw *rustcWriter) levelStyle(l Level) string {
	switch l {
	case LevelError:
		return "1;31"
	case LevelWarning:
		return "1;33"
	default:
		return "1;36"
	}
}

func (rw *rustcWriter) finding(f Finding) error {
	lvl := f.Level
	if lvl == "" {
		lvl = LevelWarning
	}
	code := rw.levelStyle(lvl)
	header := rw.paint(string(lvl), code)
	if f.RuleID != "" {
		header += rw.paint("["+f.RuleID+"]", code)
	}
	header += ": " + f.Message + "\n"
	if _, err := io.WriteString(rw.w, header); err != nil {
		return err
	}

	startLine, endLine := displayLines(f)
	loc := fmt.Sprintf("%s:%d:%d", f.File, f.Line, f.Column)
	if rw.color {
		if link := fileURL(rw.root, f.File, f.Line); link != "" {
			loc = osc8(link, loc)
		}
	}
	arrow := " " + rw.paint("-->", "1;34") + " " + loc + "\n"
	if _, err := io.WriteString(rw.w, arrow); err != nil {
		return err
	}

	lines := sourceLines(f)
	lo, hi := startLine, endLine
	if len(f.Source) > 0 {
		lo, hi = withNeighbors(startLine, endLine, displayLineCount(lines))
	}
	gw := gutterWidth(lo, hi)
	bar := rw.paint("|", "1;34")
	for ln := lo; ln <= hi; ln++ {
		raw := lineAt(lines, ln)
		codeLine := raw
		if ln >= startLine && ln <= endLine {
			from, to := spanRange(f, ln, raw)
			codeLine = underlineSpan(raw, from, to)
		}
		num := rw.paint(fmt.Sprintf("%*d", gw, ln), "1;34")
		if _, err := fmt.Fprintf(rw.w, "%s %s %s\n", num, bar, codeLine); err != nil {
			return err
		}
	}
	return rw.notes(f, gw)
}

func (rw *rustcWriter) notes(f Finding, gw int) error {
	pad := " " + strings.Repeat(" ", gw) + " " + rw.paint("=", "1;34") + " "
	if f.Fixable && f.FixSkipped {
		line := pad + rw.paint("note", "1;36") + ": fix skipped: overlap\n"
		_, err := io.WriteString(rw.w, line)
		return err
	}
	if !f.Fixable {
		return nil
	}
	line := pad + rw.paint("help", "1;32") + ": fix available\n"
	if _, err := io.WriteString(rw.w, line); err != nil {
		return err
	}
	if len(f.Source) == 0 || len(f.Edits) == 0 {
		return nil
	}
	for _, e := range f.Edits {
		if err := rw.diff(f.Source, e, gw); err != nil {
			return err
		}
	}
	return nil
}

func (rw *rustcWriter) diff(src []byte, e Edit, gw int) error {
	oldLines, newLines, startLine, ok := editLineDiff(src, e)
	if !ok {
		return nil
	}
	ln := startLine
	for _, old := range oldLines {
		num := rw.paint(fmt.Sprintf("%*d", gw, ln), "1;34")
		if _, err := fmt.Fprintf(rw.w, "%s %s %s\n", num, rw.paint("-", "31"), old); err != nil {
			return err
		}
		ln++
	}
	ln = startLine
	for _, neu := range newLines {
		num := rw.paint(fmt.Sprintf("%*d", gw, ln), "1;34")
		if _, err := fmt.Fprintf(rw.w, "%s %s %s\n", num, rw.paint("+", "32"), neu); err != nil {
			return err
		}
		ln++
	}
	return nil
}

func sourceLines(f Finding) []string {
	if len(f.Source) == 0 {
		if f.Snippet == "" {
			return nil
		}
		n := max(f.Line, 1)
		raw := make([]string, n)
		raw[n-1] = f.Snippet
		return raw
	}
	return strings.Split(string(f.Source), "\n")
}

func lineAt(lines []string, line int) string {
	i := line - 1
	if i >= 0 && i < len(lines) {
		return lines[i]
	}
	return ""
}

func displayLines(f Finding) (start, end int) {
	start = f.Line
	if start < 1 {
		start = 1
	}
	end = f.EndLine
	// A half-open span that ends at column 1 of the next line is the end of the previous line.
	if end > start && f.EndCol <= 1 {
		end--
	}
	if end < start {
		end = start
	}
	return start, end
}

func withNeighbors(start, end, maxLine int) (lo, hi int) {
	lo = start - 1
	if lo < 1 {
		lo = 1
	}
	hi = end + 1
	if maxLine > 0 && hi > maxLine {
		hi = maxLine
	}
	if hi < end {
		hi = end
	}
	return lo, hi
}

func displayLineCount(lines []string) int {
	n := len(lines)
	if n > 0 && lines[n-1] == "" {
		return n - 1
	}
	return n
}

func spanRange(f Finding, ln int, raw string) (from, to int) {
	startLine, endLine := displayLines(f)
	from, to = 1, visualWidth(raw)+1
	if ln == startLine && f.Column > 0 {
		from = f.Column
	}
	if ln == endLine && f.EndCol > 1 {
		to = f.EndCol
	}
	if to <= from {
		to = from + 1
	}
	return from, to
}

// underlineSpan wraps the 1-based byte-column range [from, to) with SGR underline.
// 24 turns underline off and leaves other attributes alone.
func underlineSpan(s string, from, to int) string {
	before, mid, after := splitANSIByCol(s, from, to)
	if mid == "" {
		return s
	}
	return before + "\x1b[4m" + keepUnderline(mid) + "\x1b[24m" + after
}

// keepUnderline re-asserts SGR 4 after every escape so a reset does not drop the underline.
func keepUnderline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			j := skipANSI(s, i)
			b.WriteString(s[i:j])
			b.WriteString("\x1b[4m")
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func splitANSIByCol(s string, from, to int) (before, mid, after string) {
	if from < 1 {
		from = 1
	}
	if to < from {
		to = from
	}
	var b, m, a strings.Builder
	col := 1
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			j := skipANSI(s, i)
			chunk := s[i:j]
			switch {
			case col < from:
				b.WriteString(chunk)
			case col < to:
				m.WriteString(chunk)
			default:
				a.WriteString(chunk)
			}
			i = j
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		chunk := s[i : i+size]
		switch {
		case col < from:
			b.WriteString(chunk)
		case col < to:
			m.WriteString(chunk)
		default:
			a.WriteString(chunk)
		}
		col += size
		i += size
	}
	return b.String(), m.String(), a.String()
}

func skipANSI(s string, i int) int {
	if i+1 >= len(s) {
		return len(s)
	}
	switch s[i+1] {
	case '[':
		j := i + 2
		for j < len(s) {
			c := s[j]
			j++
			if c >= '@' && c <= '~' {
				break
			}
		}
		return j
	case ']':
		j := i + 2
		for j < len(s) {
			if s[j] == '\a' {
				return j + 1
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2
			}
			j++
		}
		return j
	default:
		return i + 2
	}
}

func editLineDiff(src []byte, e Edit) (oldLines, newLines []string, startLine int, ok bool) {
	if int(e.EndByte) > len(src) || e.StartByte > e.EndByte {
		return nil, nil, 0, false
	}
	line, _, _, _, _, err := SpanLoc(src, e.Span())
	if err != nil {
		return nil, nil, 0, false
	}
	ls := lineStart(src, int(e.StartByte))
	le := lineEnd(src, int(e.EndByte))
	old := strings.TrimRight(string(src[ls:le]), "\n")
	prefix := string(src[ls:e.StartByte])
	suffix := strings.TrimRight(string(src[e.EndByte:le]), "\n")
	neu := prefix + e.NewText + suffix
	if old == "" && neu == "" {
		return nil, nil, 0, false
	}
	if old != "" {
		oldLines = strings.Split(old, "\n")
	}
	if neu != "" {
		newLines = strings.Split(neu, "\n")
	}
	return oldLines, newLines, line, true
}

func lineStart(src []byte, off int) int {
	if off > len(src) {
		off = len(src)
	}
	for off > 0 && src[off-1] != '\n' {
		off--
	}
	return off
}

func lineEnd(src []byte, off int) int {
	if off > len(src) {
		off = len(src)
	}
	for off < len(src) && src[off] != '\n' {
		off++
	}
	if off < len(src) {
		off++
	}
	return off
}

func gutterWidth(start, end int) int {
	n := max(start, end)
	if n < 1 {
		return 1
	}
	return len(strconv.Itoa(n))
}

func visualWidth(s string) int {
	w := 0
	for _, r := range s {
		if r == '\t' {
			w += 4 - w%4
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue
		}
		w++
	}
	return w
}

func fileURL(root, file string, line int) string {
	if file == "" {
		return ""
	}
	p := file
	if root != "" && !filepath.IsAbs(file) {
		p = path.New(root, filepath.FromSlash(file)).String()
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return ""
	}
	u := url.URL{Scheme: "file", Path: abs}
	if line > 0 {
		return u.String() + "#L" + strconv.Itoa(line)
	}
	return u.String()
}

func osc8(target, text string) string {
	return "\x1b]8;;" + target + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
