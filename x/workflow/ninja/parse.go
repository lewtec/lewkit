package ninja

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/workflow"
)

type block int

const (
	blockNone block = iota
	blockRule
	blockEdge
	blockPool
)

func (f *File) parseFile(ctx context.Context, path string, sc *scope) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := joinContinued(string(data))
	lines := strings.Split(text, "\n")
	var (
		open    block
		curRule *rule
		curEdge *edge
	)
	for n, line := range lines {
		if err := ctx.Err(); err != nil {
			return err
		}
		no := n + 1
		if strings.TrimSpace(line) == "" {
			continue
		}
		indented := line[0] == ' ' || line[0] == '\t'
		stripped := stripComment(line)
		trim := strings.TrimSpace(stripped)
		if trim == "" {
			continue
		}
		if indented {
			key, val, ok := binding(trim)
			if !ok {
				return syntax(path, no, "bad binding")
			}
			switch open {
			case blockRule:
				curRule.vars[key] = val
			case blockEdge:
				if curEdge.vars == nil {
					curEdge.vars = map[string]string{}
				}
				curEdge.vars[key] = val
			case blockPool:
			default:
				return syntax(path, no, "binding outside a block")
			}
			continue
		}
		open = blockNone
		fields := strings.Fields(trim)
		switch fields[0] {
		case "rule":
			if len(fields) != 2 {
				return syntax(path, no, "rule needs a name")
			}
			if _, ok := f.rules[fields[1]]; ok {
				return syntax(path, no, "duplicate rule "+fields[1])
			}
			curRule = &rule{vars: map[string]string{}}
			f.rules[fields[1]] = curRule
			open = blockRule
		case "build":
			edge, err := f.parseBuild(path, no, trim[len("build"):], sc)
			if err != nil {
				return err
			}
			if edge.rule != "phony" {
				if _, ok := f.rules[edge.rule]; !ok {
					return syntax(path, no, "unknown rule "+edge.rule)
				}
			}
			curEdge = edge
			f.edges = append(f.edges, edge)
			open = blockEdge
		case "default":
			rest, err := expand(strings.TrimSpace(trim[len("default"):]), sc.resolve, true)
			if err != nil {
				return fmt.Errorf("ninja: %s:%d: %w", path, no, err)
			}
			f.defaults = append(f.defaults, tokenize(rest)...)
		case "include", "subninja":
			rest := strings.TrimSpace(trim[len(fields[0]):])
			expanded, err := expand(rest, sc.resolve, true)
			if err != nil {
				return fmt.Errorf("ninja: %s:%d: %w", path, no, err)
			}
			inc := strings.TrimSpace(strings.ReplaceAll(expanded, "\x00", " "))
			inc = workflow.Join(filepath.Dir(path), inc)
			next := sc
			if fields[0] == "subninja" {
				next = &scope{parent: sc, vars: map[string]string{}}
			}
			if err := f.parseFile(ctx, inc, next); err != nil {
				return err
			}
		case "pool":
			if len(fields) != 2 {
				return syntax(path, no, "pool needs a name")
			}
			open = blockPool
		default:
			name, val, ok := binding(trim)
			if !ok {
				return syntax(path, no, "unknown statement")
			}
			sc.vars[name] = val
		}
	}
	return nil
}

func (s *scope) resolve(name string) (string, bool, bool) {
	v, ok := s.get(name)
	return v, false, ok
}

func (f *File) parseBuild(path string, no int, rest string, scope *scope) (*edge, error) {
	expanded, err := expand(strings.TrimSpace(rest), scope.resolve, true)
	if err != nil {
		return nil, fmt.Errorf("ninja: %s:%d: %w", path, no, err)
	}
	toks := tokenize(expanded)
	var outs, implicitOut []string
	dst := &outs
	i := 0
	for ; i < len(toks) && toks[i] != ":"; i++ {
		switch toks[i] {
		case "|":
			dst = &implicitOut
		case "||":
			return nil, syntax(path, no, "order-only mark before the rule")
		default:
			*dst = append(*dst, toks[i])
		}
	}
	outs = append(outs, implicitOut...)
	if i >= len(toks) || toks[i] != ":" || len(outs) == 0 {
		return nil, syntax(path, no, "build needs outputs and a rule")
	}
	i++
	if i >= len(toks) {
		return nil, syntax(path, no, "build needs a rule")
	}
	rule := toks[i]
	i++
	var explicit, implicit, order []string
	dst = &explicit
	for ; i < len(toks); i++ {
		switch toks[i] {
		case "|":
			dst = &implicit
		case "||":
			dst = &order
		default:
			*dst = append(*dst, toks[i])
		}
	}
	return &edge{
		outs:     outs,
		explicit: explicit,
		implicit: implicit,
		order:    order,
		rule:     rule,
		file:     path,
		line:     no,
	}, nil
}

func restoreToken(s string) string {
	s = strings.ReplaceAll(s, "\x00", " ")
	s = strings.ReplaceAll(s, "\x01", ":")
	s = strings.ReplaceAll(s, "\x02", "|")
	return s
}

func binding(trim string) (string, string, bool) {
	key, val, ok := strings.Cut(trim, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	if strings.HasPrefix(val, " ") {
		val = val[1:]
	}
	return key, val, true
}

func tokenize(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, restoreToken(b.String()))
		b.Reset()
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t':
			flush()
		case '|':
			flush()
			if i+1 < len(s) && s[i+1] == '|' {
				out = append(out, "||")
				i++
			} else {
				out = append(out, "|")
			}
		case ':':
			flush()
			out = append(out, ":")
		default:
			b.WriteByte(s[i])
		}
	}
	flush()
	return out
}

func joinContinued(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var b strings.Builder
	for i := 0; i < len(src); i++ {
		if src[i] == '$' && i+1 < len(src) && src[i+1] == '\n' {
			i++
			continue
		}
		b.WriteByte(src[i])
	}
	return b.String()
}

// stripComment drops a ninja comment line. A '#' later in the line is
// text: CMake writes "#include" inside command values, and ninja keeps it.
func stripComment(s string) string {
	trim := strings.TrimLeft(s, " \t")
	if strings.HasPrefix(trim, "#") {
		return ""
	}
	return s
}

func syntax(path string, line int, msg string) error {
	return fmt.Errorf("ninja: %s:%d: %w: %s", path, line, ErrSyntax, msg)
}
