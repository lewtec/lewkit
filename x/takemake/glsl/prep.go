package glsl

import (
	"fmt"
	"strings"
)

type macro struct {
	params []string
	body   string
	call   bool
}

type prepared struct {
	version    int
	profile    Profile
	extensions []Extension
	source     string
}

func prepare(src string) (prepared, error) {
	src = strings.TrimPrefix(src, "\uFEFF")
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src = spliceLines(src)
	src = stripComments(src)
	return directives(src)
}

func spliceLines(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	for i := 0; i < len(src); i++ {
		if src[i] == '\\' && i+1 < len(src) && src[i+1] == '\n' {
			i++
			continue
		}
		b.WriteByte(src[i])
	}
	return b.String()
}

func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	for i := 0; i < len(src); i++ {
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				b.WriteByte('\n')
			}
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					b.WriteByte('\n')
				}
				i++
			}
			if i+1 < len(src) {
				i++
			}
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(src[i])
	}
	return b.String()
}

type branch struct {
	parent bool
	live   bool
	taken  bool
}

func directives(src string) (prepared, error) {
	var out prepared
	macros := map[string]*macro{}
	var stack []branch
	var b strings.Builder
	lineNo := 1
	for line := range strings.SplitSeq(src, "\n") {
		if isDirective(line) {
			if err := directive(line, lineNo, &out, macros, &stack); err != nil {
				return prepared{}, err
			}
		} else if emitting(stack) {
			expanded, err := expand(line, macros, nil, 0)
			if err != nil {
				return prepared{}, err
			}
			b.WriteString(expanded)
			b.WriteByte('\n')
		}
		lineNo++
	}
	if len(stack) != 0 {
		return prepared{}, fmt.Errorf("%w: line %d: unterminated conditional", ErrSyntax, lineNo)
	}
	out.source = b.String()
	return out, nil
}

func emitting(stack []branch) bool {
	if len(stack) == 0 {
		return true
	}
	top := stack[len(stack)-1]
	return top.parent && top.live
}

func isDirective(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

func directive(line string, lineNo int, out *prepared, macros map[string]*macro, stack *[]branch) error {
	word, rest := directiveWord(line)
	switch word {
	case "version":
		if !emitting(*stack) {
			return nil
		}
		return parseVersion(rest, lineNo, out)
	case "extension":
		if !emitting(*stack) {
			return nil
		}
		return parseExtension(rest, lineNo, out)
	case "line", "pragma":
		return nil
	case "define":
		if !emitting(*stack) {
			return nil
		}
		return parseDefine(rest, lineNo, macros)
	case "undef":
		if !emitting(*stack) {
			return nil
		}
		name, _ := takeIdent(strings.TrimSpace(rest))
		if name == "" {
			return fmt.Errorf("%w: line %d: undef", ErrSyntax, lineNo)
		}
		delete(macros, name)
		return nil
	case "ifdef", "ifndef":
		name, _ := takeIdent(strings.TrimSpace(rest))
		if name == "" {
			return fmt.Errorf("%w: line %d: %s", ErrSyntax, lineNo, word)
		}
		cond := macros[name] != nil
		if word == "ifndef" {
			cond = !cond
		}
		pushBranch(stack, cond)
		return nil
	case "if":
		cond, err := ifConst(rest, macros)
		if err != nil {
			return fmt.Errorf("%w: line %d: %w", ErrSyntax, lineNo, err)
		}
		pushBranch(stack, cond)
		return nil
	case "elif":
		if len(*stack) == 0 {
			return fmt.Errorf("%w: line %d: elif", ErrSyntax, lineNo)
		}
		cond, err := ifConst(rest, macros)
		if err != nil {
			return fmt.Errorf("%w: line %d: %w", ErrSyntax, lineNo, err)
		}
		top := &(*stack)[len(*stack)-1]
		if top.taken || !top.parent {
			top.live = false
			return nil
		}
		top.live = cond
		if cond {
			top.taken = true
		}
		return nil
	case "else":
		if len(*stack) == 0 {
			return fmt.Errorf("%w: line %d: else", ErrSyntax, lineNo)
		}
		top := &(*stack)[len(*stack)-1]
		top.live = top.parent && !top.taken
		if top.live {
			top.taken = true
		}
		return nil
	case "endif":
		if len(*stack) == 0 {
			return fmt.Errorf("%w: line %d: endif", ErrSyntax, lineNo)
		}
		*stack = (*stack)[:len(*stack)-1]
		return nil
	case "include", "error":
		return fmt.Errorf("%w: line %d: #%s", ErrSyntax, lineNo, word)
	case "":
		return fmt.Errorf("%w: line %d: empty directive", ErrSyntax, lineNo)
	default:
		return fmt.Errorf("%w: line %d: #%s", ErrSyntax, lineNo, word)
	}
}

func pushBranch(stack *[]branch, cond bool) {
	parent := emitting(*stack)
	*stack = append(*stack, branch{parent: parent, live: parent && cond, taken: cond})
}

func directiveWord(line string) (string, string) {
	rest := strings.TrimSpace(line)
	rest = strings.TrimPrefix(rest, "#")
	rest = strings.TrimSpace(rest)
	name, after := takeIdent(rest)
	return name, strings.TrimSpace(after)
}

func parseVersion(rest string, lineNo int, out *prepared) error {
	if out.version != 0 {
		return fmt.Errorf("%w: line %d: version", ErrSyntax, lineNo)
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 || len(fields) > 2 {
		return fmt.Errorf("%w: line %d: version", ErrSyntax, lineNo)
	}
	var number int
	for _, r := range fields[0] {
		if r < '0' || r > '9' {
			return fmt.Errorf("%w: line %d: version", ErrSyntax, lineNo)
		}
		number = number*10 + int(r-'0')
	}
	if number == 0 {
		return fmt.Errorf("%w: line %d: version", ErrSyntax, lineNo)
	}
	out.version = number
	if len(fields) == 2 {
		profile, err := ParseProfile(fields[1])
		if err != nil {
			return fmt.Errorf("%w: line %d: version %s", ErrSyntax, lineNo, fields[1])
		}
		out.profile = profile
	}
	return nil
}

func parseExtension(rest string, lineNo int, out *prepared) error {
	name, after := takeIdent(strings.TrimSpace(rest))
	after = strings.TrimSpace(after)
	if name == "" || !strings.HasPrefix(after, ":") {
		return fmt.Errorf("%w: line %d: extension", ErrSyntax, lineNo)
	}
	behavior, err := ParseBehavior(strings.TrimSpace(after[1:]))
	if err != nil {
		return fmt.Errorf("%w: line %d: extension", ErrSyntax, lineNo)
	}
	out.extensions = append(out.extensions, Extension{Name: name, Behavior: behavior})
	return nil
}

func parseDefine(rest string, lineNo int, macros map[string]*macro) error {
	name, after := takeIdent(rest)
	if name == "" {
		return fmt.Errorf("%w: line %d: define", ErrSyntax, lineNo)
	}
	item := &macro{}
	if strings.HasPrefix(after, "(") {
		params, body, err := defineParams(after)
		if err != nil {
			return fmt.Errorf("%w: line %d: define %s", ErrSyntax, lineNo, name)
		}
		item.call = true
		item.params = params
		item.body = body
	} else {
		item.body = strings.TrimSpace(after)
	}
	macros[name] = item
	return nil
}

func defineParams(rest string) ([]string, string, error) {
	if !strings.HasPrefix(rest, "(") {
		return nil, "", fmt.Errorf("%w: define", ErrSyntax)
	}
	end := strings.IndexByte(rest, ')')
	if end < 0 {
		return nil, "", fmt.Errorf("%w: define", ErrSyntax)
	}
	inside := strings.TrimSpace(rest[1:end])
	var params []string
	if inside != "" {
		for _, part := range strings.Split(inside, ",") {
			param, extra := takeIdent(strings.TrimSpace(part))
			if param == "" || strings.TrimSpace(extra) != "" {
				return nil, "", fmt.Errorf("%w: define", ErrSyntax)
			}
			params = append(params, param)
		}
	}
	return params, strings.TrimSpace(rest[end+1:]), nil
}

func ifConst(rest string, macros map[string]*macro) (bool, error) {
	expanded, err := expandDefined(rest, macros)
	if err != nil {
		return false, err
	}
	value, err := evalIf(expanded)
	if err != nil {
		return false, err
	}
	return value != 0, nil
}

func expandDefined(rest string, macros map[string]*macro) (string, error) {
	var b strings.Builder
	for i := 0; i < len(rest); {
		if isIdentStart(rest[i]) {
			name, after := takeIdent(rest[i:])
			if name == "defined" {
				id, next, err := definedOperand(after)
				if err != nil {
					return "", err
				}
				if macros[id] != nil {
					b.WriteByte('1')
				} else {
					b.WriteByte('0')
				}
				i += len(rest[i:]) - len(next)
				continue
			}
			if item := macros[name]; item != nil && !item.call {
				b.WriteString(item.body)
				i += len(name)
				continue
			}
			b.WriteString(name)
			i += len(name)
			continue
		}
		b.WriteByte(rest[i])
		i++
	}
	return b.String(), nil
}

func definedOperand(rest string) (string, string, error) {
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "(") {
		rest = strings.TrimSpace(rest[1:])
		name, after := takeIdent(rest)
		after = strings.TrimSpace(after)
		if name == "" || !strings.HasPrefix(after, ")") {
			return "", "", fmt.Errorf("%w: defined", ErrSyntax)
		}
		return name, after[1:], nil
	}
	name, after := takeIdent(rest)
	if name == "" {
		return "", "", fmt.Errorf("%w: defined", ErrSyntax)
	}
	return name, after, nil
}

func evalIf(src string) (int64, error) {
	parser := &ifParser{src: src}
	value, err := parser.or()
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(parser.src[parser.i:]) != "" {
		return 0, fmt.Errorf("%w: #if", ErrSyntax)
	}
	return value, nil
}

type ifParser struct {
	src string
	i   int
}

func (parser *ifParser) or() (int64, error) {
	left, err := parser.and()
	if err != nil {
		return 0, err
	}
	for parser.has("||") {
		parser.i += 2
		right, err := parser.and()
		if err != nil {
			return 0, err
		}
		if left != 0 || right != 0 {
			left = 1
		} else {
			left = 0
		}
	}
	return left, nil
}

func (parser *ifParser) and() (int64, error) {
	left, err := parser.not()
	if err != nil {
		return 0, err
	}
	for parser.has("&&") {
		parser.i += 2
		right, err := parser.not()
		if err != nil {
			return 0, err
		}
		if left != 0 && right != 0 {
			left = 1
		} else {
			left = 0
		}
	}
	return left, nil
}

func (parser *ifParser) not() (int64, error) {
	parser.skip()
	if parser.has("!") {
		parser.i++
		value, err := parser.not()
		if err != nil {
			return 0, err
		}
		if value == 0 {
			return 1, nil
		}
		return 0, nil
	}
	return parser.primary()
}

func (parser *ifParser) primary() (int64, error) {
	parser.skip()
	if parser.i >= len(parser.src) {
		return 0, fmt.Errorf("%w: #if", ErrSyntax)
	}
	if parser.src[parser.i] == '(' {
		parser.i++
		value, err := parser.or()
		if err != nil {
			return 0, err
		}
		parser.skip()
		if parser.i >= len(parser.src) || parser.src[parser.i] != ')' {
			return 0, fmt.Errorf("%w: #if", ErrSyntax)
		}
		parser.i++
		return value, nil
	}
	if parser.src[parser.i] >= '0' && parser.src[parser.i] <= '9' {
		start := parser.i
		for parser.i < len(parser.src) && parser.src[parser.i] >= '0' && parser.src[parser.i] <= '9' {
			parser.i++
		}
		var value int64
		for _, r := range parser.src[start:parser.i] {
			value = value*10 + int64(r-'0')
		}
		return value, nil
	}
	if isIdentStart(parser.src[parser.i]) {
		_, after := takeIdent(parser.src[parser.i:])
		parser.i = len(parser.src) - len(after)
		return 0, nil
	}
	return 0, fmt.Errorf("%w: #if", ErrSyntax)
}

func (parser *ifParser) has(op string) bool {
	parser.skip()
	return strings.HasPrefix(parser.src[parser.i:], op)
}

func (parser *ifParser) skip() {
	for parser.i < len(parser.src) && (parser.src[parser.i] == ' ' || parser.src[parser.i] == '\t') {
		parser.i++
	}
}

func expand(src string, macros map[string]*macro, hidden map[string]bool, depth int) (string, error) {
	if depth > 64 {
		return "", fmt.Errorf("%w: macro depth", ErrSyntax)
	}
	var b strings.Builder
	for i := 0; i < len(src); {
		if isIdentStart(src[i]) {
			name, _ := takeIdent(src[i:])
			item := macros[name]
			if item != nil && !hidden[name] {
				nextHidden := hide(hidden, name)
				if item.call {
					rest := src[i+len(name):]
					space := len(rest) - len(strings.TrimLeft(rest, " \t"))
					if space < len(rest) && rest[space] == '(' {
						args, offset, err := readArgs(rest[space:])
						if err != nil {
							return "", err
						}
						repl, err := applyMacro(item, args, macros, nextHidden, depth)
						if err != nil {
							return "", err
						}
						b.WriteString(repl)
						i += len(name) + space + offset
						continue
					}
				} else {
					repl, err := expand(item.body, macros, nextHidden, depth+1)
					if err != nil {
						return "", err
					}
					b.WriteString(repl)
					i += len(name)
					continue
				}
			}
			b.WriteString(name)
			i += len(name)
			continue
		}
		b.WriteByte(src[i])
		i++
	}
	return b.String(), nil
}

func hide(hidden map[string]bool, name string) map[string]bool {
	out := map[string]bool{name: true}
	for key := range hidden {
		out[key] = true
	}
	return out
}

func applyMacro(item *macro, args []string, macros map[string]*macro, hidden map[string]bool, depth int) (string, error) {
	if len(args) != len(item.params) {
		return "", fmt.Errorf("%w: macro arguments", ErrSyntax)
	}
	expanded := make([]string, len(args))
	for i, arg := range args {
		text, err := expand(arg, macros, hidden, depth+1)
		if err != nil {
			return "", err
		}
		expanded[i] = text
	}
	body := substitute(item.body, item.params, expanded)
	return expand(body, macros, hidden, depth+1)
}

func substitute(body string, params []string, args []string) string {
	index := map[string]string{}
	for i, param := range params {
		index[param] = args[i]
	}
	var b strings.Builder
	for i := 0; i < len(body); {
		if isIdentStart(body[i]) {
			name, _ := takeIdent(body[i:])
			if arg, ok := index[name]; ok {
				b.WriteString(arg)
			} else {
				b.WriteString(name)
			}
			i += len(name)
			continue
		}
		b.WriteByte(body[i])
		i++
	}
	return b.String()
}

func readArgs(src string) ([]string, int, error) {
	if !strings.HasPrefix(src, "(") {
		return nil, 0, fmt.Errorf("%w: macro", ErrSyntax)
	}
	depth := 0
	start := 1
	var args []string
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				inside := strings.TrimSpace(src[start:i])
				if len(args) == 0 && inside == "" {
					return nil, i + 1, nil
				}
				args = append(args, inside)
				return args, i + 1, nil
			}
		case ',':
			if depth == 1 {
				args = append(args, strings.TrimSpace(src[start:i]))
				start = i + 1
			}
		}
	}
	return nil, 0, fmt.Errorf("%w: macro", ErrSyntax)
}

func takeIdent(src string) (string, string) {
	if src == "" || !isIdentStart(src[0]) {
		return "", src
	}
	i := 1
	for i < len(src) && isIdentCont(src[i]) {
		i++
	}
	return src[:i], src[i:]
}

func isIdentStart(b byte) bool {
	return b == '_' || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isIdentCont(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}
