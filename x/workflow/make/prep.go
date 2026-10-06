package make

import (
	"encoding/hex"
	"strings"
)

// refOpen and refClose hide a $(...) or ${...} inside an assignment name.
// The make grammar only accepts a single word on the left of an assignment,
// so obj-$(CONFIG_FOO) is read as a rule. The markers are not make syntax.
const (
	refOpen              = "_lewref_"
	refClose             = "_wer_"
	targetAssignName     = "lewkit_target_assign"
	rawAssignName        = "lewkit_raw_assign"
	emptyImmediateExpand = "$(subst x,x,)"
)

// targetVar is an assignment that applies while one target's recipe runs.
type targetVar struct {
	targets []string
	name    string
	op      string
	value   string
	folded  bool
	export  bool
	private bool
	origin  origin
}

// prepareSrc rewrites makefile text the tree-sitter grammar misreads.
// Empty VPATH assignments swallow the next line. A name that contains a
// variable reference is parsed as a rule. Target-specific assignments are
// not in the grammar; they are carried as lewkit_target_assign.
// An empty substitution reference such as $(dirs:/=) is a syntax error;
// a non-empty replacement parses, and $(subst x,x,) expands to nothing.
func prepareSrc(src string) string {
	src = rewriteEmptySubst(src)
	lines := strings.Split(src, "\n")
	depth := 0
	inRecipe := false
	for i := 0; i < len(lines); {
		line := lines[i]
		if depth > 0 {
			if isEndef(strings.TrimSpace(stripMakeComment(line))) {
				depth--
			}
			i++
			continue
		}
		// A tab outside a rule is how the kernel writes assignments inside
		// ifeq. Inside a rule the same tab is a recipe and must stay.
		if strings.HasPrefix(line, "\t") {
			if inRecipe {
				i++
				continue
			}
			stripped, ok := untabStatement(line)
			if !ok {
				i++
				continue
			}
			line = stripped
			lines[i] = line
		}
		trim := strings.TrimSpace(stripMakeComment(line))
		if trim == "" {
			i++
			continue
		}
		if isDefine(trim) {
			depth++
			i++
			continue
		}
		if !directiveLine(trim) {
			inRecipe = opensRecipe(line)
		}
		logical, extra := joinContinued(lines, i)
		rewritten, ok := rewriteMakeLine(logical)
		if !ok {
			i++
			continue
		}
		lines[i] = rewritten
		for k := 1; k <= extra && i+k < len(lines); k++ {
			lines[i+k] = ""
		}
		i += extra + 1
	}
	return strings.Join(lines, "\n")
}

// untabStatement drops a leading tab from an assignment, directive, or comment.
// A tabbed rule stays tabbed so it is still a recipe line.
func untabStatement(line string) (string, bool) {
	rest := strings.TrimLeft(line, "\t")
	code := rest
	if hash := indexMakeComment(rest); hash >= 0 {
		code = rest[:hash]
	}
	trim := strings.TrimSpace(code)
	// A tabbed comment or blank line outside a rule is not a recipe.
	if trim == "" {
		return rest, true
	}
	if directiveLine(trim) {
		return rest, true
	}
	idx, _, colon := findSplit(code)
	if idx < 0 {
		return line, false
	}
	if !colon {
		return rest, true
	}
	if _, ok := targetAssign(code[idx+1:]); ok {
		return rest, true
	}
	return line, false
}

func opensRecipe(line string) bool {
	code := line
	if hash := indexMakeComment(line); hash >= 0 {
		code = line[:hash]
	}
	idx, _, colon := findSplit(code)
	if !colon {
		return false
	}
	_, ok := targetAssign(code[idx+1:])
	return !ok
}

func joinContinued(lines []string, start int) (string, int) {
	var b strings.Builder
	n := 0
	for {
		line := lines[start+n]
		if !continuedLine(line) {
			b.WriteString(line)
			return b.String(), n
		}
		trimmed := strings.TrimRight(line, " \t")
		trimmed = trimmed[:len(trimmed)-1]
		b.WriteString(strings.TrimRight(trimmed, " \t"))
		b.WriteByte(' ')
		n++
		if start+n >= len(lines) {
			return b.String(), n - 1
		}
	}
}

func continuedLine(line string) bool {
	line = strings.TrimRight(line, " \t")
	n := 0
	for len(line) > 0 && strings.HasSuffix(line, "\\") {
		n++
		line = line[:len(line)-1]
	}
	return n%2 == 1
}

func rewriteMakeLine(line string) (string, bool) {
	hash := indexMakeComment(line)
	code := line
	comment := ""
	if hash >= 0 {
		code = line[:hash]
		comment = line[hash:]
	}
	if strings.TrimSpace(code) == "" {
		return line, false
	}
	trim := strings.TrimSpace(code)
	if directiveLine(trim) {
		return line, false
	}
	idx, op, colon := findSplit(code)
	if idx < 0 {
		return line, false
	}
	lead := code[:len(code)-len(strings.TrimLeft(code, " "))]
	if colon {
		rec, ok := targetAssign(code[idx+1:])
		if !ok {
			return line, false
		}
		rec.targets = strings.TrimSpace(code[len(lead):idx])
		blob := encodeTarget(rec)
		return lead + targetAssignName + " := " + blob + commentPad(comment), true
	}
	lhs := code[:idx]
	value := code[idx+len(op):]
	if strings.TrimSpace(lhs) == "VPATH" && strings.TrimSpace(value) == "" {
		return strings.TrimRight(code, " \t") + " " + emptyImmediateExpand + commentPad(comment), true
	}
	if rawValue(value) {
		blob := encodeRaw(lhs, op, strings.TrimSpace(value))
		return lead + rawAssignName + " := " + blob + commentPad(comment), true
	}
	enc := encodeRefs(lhs)
	if enc == lhs {
		return line, false
	}
	return enc + op + value + comment, true
}

// rawValue reports assignment text the grammar splits on. A shell pipeline
// written as $$(...) is literal, but a nested $(...) before the pipe makes
// the parser treat the pipe as syntax.
func rawValue(value string) bool {
	return strings.Contains(value, "$$(") && strings.Contains(value, "|")
}

// rewriteEmptySubst fills an empty $(name:pat=) so the grammar keeps it.
// $$ stays a literal dollar, and a space in the replacement is kept:
// GNU make treats $(dirs:/= ) as "replace with a space", not empty.
func rewriteEmptySubst(src string) string {
	var b strings.Builder
	b.Grow(len(src) + 16)
	for i := 0; i < len(src); {
		if src[i] != '$' {
			b.WriteByte(src[i])
			i++
			continue
		}
		j := i
		for j < len(src) && src[j] == '$' {
			j++
		}
		n := j - i
		for k := 0; k < n/2; k++ {
			b.WriteString("$$")
		}
		if n%2 == 0 {
			i = j
			continue
		}
		if j >= len(src) || (src[j] != '(' && src[j] != '{') {
			b.WriteByte('$')
			i = j
			continue
		}
		open := src[j]
		closer := byte(')')
		if open == '{' {
			closer = '}'
		}
		body, end, err := scanClose(src, j, open, closer)
		if err != nil {
			b.WriteByte('$')
			i = j
			continue
		}
		body = rewriteEmptySubst(body)
		b.WriteByte('$')
		b.WriteByte(open)
		b.WriteString(fillEmptySubst(body))
		b.WriteByte(closer)
		i = end + 1
	}
	return b.String()
}

func fillEmptySubst(body string) string {
	if name, rest, ok := cutFunc(body); ok {
		if !commaFunc(name) {
			return body
		}
		return name + " " + fillEmptyArgs(rest)
	}
	name, pat, rep, ok := cutSubst(body)
	if !ok || rep != "" || pat == "" || name == "" || strings.ContainsAny(name, " \t") {
		return body
	}
	return name + ":" + pat + "=" + emptyImmediateExpand
}

// commaFunc reports functions whose commas separate arguments.
// $(shell) keeps commas in the script, so it is not one of these.
func commaFunc(name string) bool {
	switch name {
	case "if", "or", "and", "call", "foreach",
		"subst", "patsubst", "filter", "filter-out", "findstring",
		"wordlist", "join", "addprefix", "addsuffix":
		return true
	default:
		return false
	}
}

// fillEmptyArgs gives an empty argument a value the grammar accepts.
// $(if $(libs),$(dest),) is legal GNU make and means an empty else.
func fillEmptyArgs(rest string) string {
	parts := splitComma(rest)
	empty := false
	for _, part := range parts {
		if part == "" {
			empty = true
			break
		}
	}
	if !empty {
		return rest
	}
	for i, part := range parts {
		if part == "" {
			parts[i] = emptyImmediateExpand
		}
	}
	return strings.Join(parts, ",")
}

func commentPad(comment string) string {
	if comment == "" || strings.HasPrefix(comment, " ") || strings.HasPrefix(comment, "\t") {
		return comment
	}
	return " " + comment
}

type targetRec struct {
	targets string
	name    string
	op      string
	value   string
	export  bool
	private bool
	over    bool
}

func targetAssign(rhs string) (targetRec, bool) {
	s := strings.TrimLeft(rhs, " \t")
	var rec targetRec
	for {
		kw, rest, ok := cutKeyword(s, "private", "export", "override")
		if !ok {
			break
		}
		switch kw {
		case "private":
			rec.private = true
		case "export":
			rec.export = true
		case "override":
			rec.over = true
		}
		s = strings.TrimLeft(rest, " \t")
	}
	idx, op, colon := findSplit(s)
	if idx <= 0 || colon || op == "" || !singleToken(s[:idx]) {
		return targetRec{}, false
	}
	rec.name = strings.TrimSpace(s[:idx])
	rec.op = op
	rec.value = strings.TrimSpace(s[idx+len(op):])
	if rec.name == "" {
		return targetRec{}, false
	}
	return rec, true
}

func cutKeyword(s string, words ...string) (string, string, bool) {
	for _, w := range words {
		if s == w || (strings.HasPrefix(s, w) && boundary(s, len(w))) {
			return w, s[len(w):], true
		}
	}
	return "", s, false
}

func singleToken(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	depth := 0
	closers := make([]byte, 0, 2)
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '$' && i+1 < len(name) && (name[i+1] == '(' || name[i+1] == '{') {
			if name[i+1] == '(' {
				closers = append(closers, ')')
			} else {
				closers = append(closers, '}')
			}
			depth++
			i++
			continue
		}
		if depth > 0 {
			if c == closers[len(closers)-1] {
				closers = closers[:len(closers)-1]
				depth--
			}
			continue
		}
		if c == ' ' || c == '\t' || c == ':' {
			return false
		}
	}
	return depth == 0
}

func findSplit(code string) (idx int, op string, colon bool) {
	depth := 0
	closers := make([]byte, 0, 4)
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c == '\\' && i+1 < len(code) {
			i++
			continue
		}
		if c == '$' && i+1 < len(code) && (code[i+1] == '(' || code[i+1] == '{') {
			if code[i+1] == '(' {
				closers = append(closers, ')')
			} else {
				closers = append(closers, '}')
			}
			depth++
			i++
			continue
		}
		if depth > 0 {
			if c == closers[len(closers)-1] {
				closers = closers[:len(closers)-1]
				depth--
			}
			continue
		}
		if got, n := operatorAt(code, i); n > 0 {
			return i, got, false
		}
		if c == ':' {
			return i, "", true
		}
	}
	return -1, "", false
}

func operatorAt(s string, i int) (string, int) {
	for _, op := range []string{"::=", ":=", "!=", "?=", "+=", "="} {
		if strings.HasPrefix(s[i:], op) {
			return op, len(op)
		}
	}
	return "", 0
}

func directiveLine(trim string) bool {
	switch {
	case trim == "define" || hasWord(trim, "define"):
		return true
	case trim == "endef" || hasWord(trim, "endef"):
		return true
	case hasWord(trim, "ifeq") || hasWord(trim, "ifneq") || hasWord(trim, "ifdef") || hasWord(trim, "ifndef"):
		return true
	case trim == "else" || hasWord(trim, "else") || trim == "endif" || hasWord(trim, "endif"):
		return true
	case hasWord(trim, "include") || hasWord(trim, "-include") || hasWord(trim, "sinclude"):
		return true
	case hasWord(trim, "vpath") || hasWord(trim, "undefine") || hasWord(trim, "unexport"):
		return true
	default:
		return false
	}
}

func hasWord(s, w string) bool {
	return s == w || (strings.HasPrefix(s, w) && boundary(s, len(w)))
}

func isDefine(trim string) bool {
	return trim == "define" || hasWord(trim, "define")
}

func isEndef(trim string) bool {
	return trim == "endef" || hasWord(trim, "endef")
}

func indexMakeComment(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			continue
		}
		if i > 0 && line[i-1] == '\\' {
			continue
		}
		return i
	}
	return -1
}

func stripMakeComment(line string) string {
	if i := indexMakeComment(line); i >= 0 {
		return line[:i]
	}
	return line
}

func encodeRefs(name string) string {
	var b strings.Builder
	for i := 0; i < len(name); {
		if name[i] == '$' && i+1 < len(name) && (name[i+1] == '(' || name[i+1] == '{') {
			open := name[i+1]
			close := byte(')')
			if open == '{' {
				close = '}'
			}
			body, next, err := scanClose(name, i+1, open, close)
			if err != nil {
				b.WriteByte(name[i])
				i++
				continue
			}
			b.WriteString(refOpen)
			b.WriteString(hex.EncodeToString([]byte(body)))
			b.WriteString(refClose)
			i = next + 1
			continue
		}
		b.WriteByte(name[i])
		i++
	}
	return b.String()
}

func decodeRefs(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, refOpen)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		rest := s[i+len(refOpen):]
		j := strings.Index(rest, refClose)
		if j < 0 {
			b.WriteString(s[i:])
			return b.String()
		}
		raw, err := hex.DecodeString(rest[:j])
		if err != nil {
			b.WriteString(refOpen)
			s = rest
			continue
		}
		b.WriteByte('$')
		b.WriteByte('(')
		b.Write(raw)
		b.WriteByte(')')
		s = rest[j+len(refClose):]
	}
}

func encodeRaw(lhs, op, value string) string {
	export, over, name := splitLHS(lhs)
	flags := ""
	if export {
		flags += "e"
	}
	if over {
		flags += "o"
	}
	raw := strings.Join([]string{encodeRefs(name), op, value, flags}, "\x1e")
	return hex.EncodeToString([]byte(raw))
}

func decodeRaw(blob string) (name, op, value string, export, over bool, ok bool) {
	raw, err := hex.DecodeString(strings.TrimSpace(blob))
	if err != nil {
		return "", "", "", false, false, false
	}
	parts := strings.Split(string(raw), "\x1e")
	if len(parts) != 4 {
		return "", "", "", false, false, false
	}
	return parts[0], parts[1], parts[2], strings.Contains(parts[3], "e"), strings.Contains(parts[3], "o"), true
}

func splitLHS(lhs string) (export, over bool, name string) {
	s := strings.TrimSpace(lhs)
	for {
		kw, rest, ok := cutKeyword(s, "export", "override", "private")
		if !ok {
			break
		}
		switch kw {
		case "export":
			export = true
		case "override":
			over = true
		}
		s = strings.TrimSpace(rest)
	}
	return export, over, s
}

func encodeTarget(rec targetRec) string {
	flags := ""
	if rec.export {
		flags += "e"
	}
	if rec.private {
		flags += "p"
	}
	if rec.over {
		flags += "o"
	}
	raw := strings.Join([]string{rec.targets, rec.name, rec.op, rec.value, flags}, "\x1e")
	return hex.EncodeToString([]byte(raw))
}

func (f *File) withTargetVars(target string, fn func() error) error {
	type snap struct {
		name    string
		v       *variable
		had     bool
		exp, un bool
		hadE    bool
		hadU    bool
	}
	var snaps []snap
	seen := map[string]bool{}
	defer func() {
		for i := len(snaps) - 1; i >= 0; i-- {
			s := snaps[i]
			if s.had {
				f.vars[s.name] = s.v
			} else {
				delete(f.vars, s.name)
			}
			if s.hadE {
				f.exported[s.name] = s.exp
			} else {
				delete(f.exported, s.name)
			}
			if s.hadU {
				f.unexported[s.name] = s.un
			} else {
				delete(f.unexported, s.name)
			}
		}
	}()
	for _, tv := range f.targetVars {
		if !matchTargetList(tv.targets, target) {
			continue
		}
		if !seen[tv.name] {
			seen[tv.name] = true
			s := snap{name: tv.name}
			if v, ok := f.vars[tv.name]; ok {
				cp := *v
				s.v = &cp
				s.had = true
			}
			s.exp = f.exported[tv.name]
			s.hadE = s.exp
			s.un = f.unexported[tv.name]
			s.hadU = s.un
			snaps = append(snaps, s)
		}
		if tv.folded {
			f.vars[tv.name] = &variable{value: tv.value, simple: true, origin: tv.origin}
		} else if err := f.assign(tv.name, tv.value, tv.op, tv.origin); err != nil {
			return err
		}
		if tv.private {
			delete(f.exported, tv.name)
			f.unexported[tv.name] = true
		}
		if tv.export {
			f.exported[tv.name] = true
			delete(f.unexported, tv.name)
		}
	}
	return fn()
}

func matchTargetList(pats []string, name string) bool {
	for _, pat := range pats {
		if pat == name {
			return true
		}
		if strings.Contains(pat, "%") {
			if _, ok := stemOf(pat, name); ok {
				return true
			}
		}
	}
	return false
}

func decodeTarget(blob string) (targetRec, bool) {
	raw, err := hex.DecodeString(strings.TrimSpace(blob))
	if err != nil {
		return targetRec{}, false
	}
	parts := strings.Split(string(raw), "\x1e")
	if len(parts) != 5 {
		return targetRec{}, false
	}
	rec := targetRec{targets: parts[0], name: parts[1], op: parts[2], value: parts[3]}
	rec.export = strings.Contains(parts[4], "e")
	rec.private = strings.Contains(parts[4], "p")
	rec.over = strings.Contains(parts[4], "o")
	return rec, true
}
