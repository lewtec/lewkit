package ninja

import (
	"fmt"
	"io"
	"strings"

	syntax "github.com/lewtec/lewkit/x/parse/ninja"
)

func writeFile(w io.Writer, f *File) error {
	for _, s := range f.Stmts {
		var err error
		switch n := s.(type) {
		case *Variable:
			err = writeVariable(w, n)
		case *Rule:
			err = writeRule(w, n)
		case *Edge:
			err = writeEdge(w, n)
		case *Default:
			err = writeDefault(w, n)
		case *Pool:
			err = writePool(w, n)
		case *Include:
			err = writeInclude(w, n)
		case nil:
			err = fmt.Errorf("%w: nil statement", errFile)
		default:
			err = fmt.Errorf("%w: unknown statement", errFile)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writeVariable(w io.Writer, v *Variable) error {
	if v == nil || !syntax.Ident(v.Name) || syntax.Keyword(v.Name) {
		return fmt.Errorf("%w: bad variable", errFile)
	}
	var b strings.Builder
	b.WriteString(v.Name)
	b.WriteString(" = ")
	if err := writePieces(&b, v.Value, false); err != nil {
		return err
	}
	return writeRaw(w, b.String())
}

func writeRule(w io.Writer, r *Rule) error {
	if r == nil || !syntax.Ident(r.Name) {
		return fmt.Errorf("%w: bad rule", errFile)
	}
	if err := writeRaw(w, "rule "+r.Name); err != nil {
		return err
	}
	return writeBindings(w, r.Bindings)
}

func writePool(w io.Writer, p *Pool) error {
	if p == nil || !syntax.Ident(p.Name) {
		return fmt.Errorf("%w: bad pool", errFile)
	}
	if err := writeRaw(w, "pool "+p.Name); err != nil {
		return err
	}
	return writeBindings(w, p.Bindings)
}

func writeEdge(w io.Writer, e *Edge) error {
	if e == nil || !syntax.Ident(e.Rule) {
		return fmt.Errorf("%w: bad build", errFile)
	}
	if len(e.Outputs) == 0 && len(e.ImplicitOutputs) == 0 {
		return fmt.Errorf("%w: empty build", errFile)
	}
	var b strings.Builder
	b.WriteString("build ")
	if err := writePaths(&b, e.Outputs); err != nil {
		return err
	}
	if len(e.ImplicitOutputs) > 0 {
		if len(e.Outputs) > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("| ")
		if err := writePaths(&b, e.ImplicitOutputs); err != nil {
			return err
		}
	}
	b.WriteString(": ")
	b.WriteString(e.Rule)
	if len(e.Inputs) > 0 {
		b.WriteByte(' ')
		if err := writePaths(&b, e.Inputs); err != nil {
			return err
		}
	}
	if len(e.ImplicitDeps) > 0 {
		b.WriteString(" | ")
		if err := writePaths(&b, e.ImplicitDeps); err != nil {
			return err
		}
	}
	if len(e.OrderOnlyDeps) > 0 {
		b.WriteString(" || ")
		if err := writePaths(&b, e.OrderOnlyDeps); err != nil {
			return err
		}
	}
	if len(e.Validations) > 0 {
		b.WriteString(" |@ ")
		if err := writePaths(&b, e.Validations); err != nil {
			return err
		}
	}
	if err := writeRaw(w, b.String()); err != nil {
		return err
	}
	return writeBindings(w, e.Bindings)
}

func writeDefault(w io.Writer, d *Default) error {
	if d == nil || len(d.Targets) == 0 {
		return fmt.Errorf("%w: empty default", errFile)
	}
	var b strings.Builder
	b.WriteString("default ")
	if err := writePaths(&b, d.Targets); err != nil {
		return err
	}
	return writeRaw(w, b.String())
}

func writeInclude(w io.Writer, inc *Include) error {
	if inc == nil || len(inc.Path) == 0 {
		return fmt.Errorf("%w: empty include", errFile)
	}
	word := "include"
	if inc.Subninja {
		word = "subninja"
	}
	var b strings.Builder
	b.WriteString(word)
	b.WriteByte(' ')
	if err := writePieces(&b, inc.Path, true); err != nil {
		return err
	}
	return writeRaw(w, b.String())
}

func writeBindings(w io.Writer, bs []Binding) error {
	for _, b := range bs {
		if !syntax.Ident(b.Name) {
			return fmt.Errorf("%w: bad binding", errFile)
		}
		var buf strings.Builder
		buf.WriteString("  ")
		buf.WriteString(b.Name)
		buf.WriteString(" = ")
		if err := writePieces(&buf, b.Value, false); err != nil {
			return err
		}
		if err := writeRaw(w, buf.String()); err != nil {
			return err
		}
	}
	return nil
}

func writePaths(b *strings.Builder, paths [][]Piece) error {
	for i, p := range paths {
		if len(p) == 0 {
			return fmt.Errorf("%w: empty path", errFile)
		}
		if i > 0 {
			b.WriteByte(' ')
		}
		if err := writePieces(b, p, true); err != nil {
			return err
		}
	}
	return nil
}

func braceVar(name string, rest []Piece) bool {
	if !syntax.Simple(name) {
		return true
	}
	if len(rest) == 0 || rest[0].Text == "" {
		return false
	}
	return syntax.SimpleChar(rest[0].Text[0])
}

func writePieces(b *strings.Builder, ps []Piece, path bool) error {
	for i, p := range ps {
		if (p.Text != "") == (p.Var != "") {
			return fmt.Errorf("%w: bad piece", errFile)
		}
		if p.Var != "" {
			if !syntax.Ident(p.Var) {
				return fmt.Errorf("%w: bad variable", errFile)
			}
			// $name glues to a following letter, so ${name} is required
			// when the next piece would extend the identifier.
			if braceVar(p.Var, ps[i+1:]) {
				b.WriteString("${")
				b.WriteString(p.Var)
				b.WriteByte('}')
			} else {
				b.WriteByte('$')
				b.WriteString(p.Var)
			}
			continue
		}
		if strings.ContainsAny(p.Text, "\n\r") {
			return fmt.Errorf("%w: newline in value", errFile)
		}
		if path && strings.Contains(p.Text, "|") {
			return fmt.Errorf("%w: pipe in path", errFile)
		}
		for i := 0; i < len(p.Text); i++ {
			c := p.Text[i]
			switch {
			case c == '$':
				b.WriteString("$$")
			case path && c == ' ':
				b.WriteString("$ ")
			case path && c == ':':
				b.WriteString("$:")
			default:
				b.WriteByte(c)
			}
		}
	}
	return nil
}

func writeRaw(w io.Writer, line string) error {
	if _, err := io.WriteString(w, line); err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	return nil
}
