package makefile

import (
	"fmt"
	"io"
	"strings"
)

func writeFile(w io.Writer, f *File) error {
	for _, stmt := range f.Stmts {
		if err := writeStmt(w, stmt); err != nil {
			return err
		}
	}
	return nil
}

func writeStmt(w io.Writer, stmt Stmt) error {
	switch s := stmt.(type) {
	case *Variable:
		return writeVariable(w, s)
	case *Rule:
		return writeRule(w, s)
	case *Function:
		return writeFunction(w, s)
	case *Conditional:
		return writeConditional(w, s)
	case *Include:
		return writeInclude(w, s)
	case *Export:
		return writeExport(w, s)
	case *VPath:
		return writeVPath(w, s)
	case *Undefine:
		return writeUndefine(w, s)
	default:
		return fmt.Errorf("%w: nil statement", errFile)
	}
}

func writeVariable(w io.Writer, v *Variable) error {
	if v == nil || v.Name == "" {
		return fmt.Errorf("%w: empty variable name", errFile)
	}
	sym, ok := v.Op.symbol()
	if !ok {
		return fmt.Errorf("%w: variable %s operator", errSyntax, v.Name)
	}
	var b strings.Builder
	if len(v.Targets) > 0 {
		b.WriteString(strings.Join(v.Targets, " "))
		b.WriteString(": ")
	}
	writeMods(&b, v.Override, v.Export, v.Private)
	b.WriteString(v.Name)
	b.WriteByte(' ')
	b.WriteString(sym)
	b.WriteByte(' ')
	b.WriteString(v.Value)
	return writeLine(w, b.String())
}

func writeRule(w io.Writer, r *Rule) error {
	if r == nil {
		return fmt.Errorf("%w: nil rule", errFile)
	}
	if len(r.Targets) == 0 && len(r.Recipe) == 0 {
		return fmt.Errorf("%w: empty rule", errFile)
	}
	if len(r.Targets) > 0 {
		var b strings.Builder
		b.WriteString(strings.Join(r.Targets, " "))
		switch {
		case r.Grouped && r.DoubleColon:
			b.WriteString("&::")
		case r.Grouped:
			b.WriteString("&:")
		case r.DoubleColon:
			b.WriteString("::")
		default:
			b.WriteByte(':')
		}
		if r.Pattern != "" {
			b.WriteByte(' ')
			b.WriteString(r.Pattern)
			b.WriteByte(':')
		}
		if len(r.Prerequisites) > 0 || len(r.OrderOnly) > 0 {
			b.WriteByte(' ')
			b.WriteString(strings.Join(r.Prerequisites, " "))
			if len(r.OrderOnly) > 0 {
				if len(r.Prerequisites) > 0 {
					b.WriteByte(' ')
				}
				b.WriteString("| ")
				b.WriteString(strings.Join(r.OrderOnly, " "))
			}
		}
		if err := writeLine(w, b.String()); err != nil {
			return err
		}
	}
	for _, line := range r.Recipe {
		if strings.Contains(line, "\n") {
			return fmt.Errorf("%w: recipe line contains a newline", errSyntax)
		}
		if _, err := io.WriteString(w, "\t"+line+"\n"); err != nil {
			return fmt.Errorf("%w: %w", errWrite, err)
		}
	}
	return nil
}

func writeFunction(w io.Writer, fn *Function) error {
	if fn == nil || fn.Name == "" {
		return fmt.Errorf("%w: empty variable name", errFile)
	}
	var b strings.Builder
	writeMods(&b, fn.Override, fn.Export, fn.Private)
	b.WriteString("define ")
	b.WriteString(fn.Name)
	if fn.Op != 0 {
		sym, ok := fn.Op.symbol()
		if !ok {
			return fmt.Errorf("%w: define %s operator", errSyntax, fn.Name)
		}
		b.WriteByte(' ')
		b.WriteString(sym)
	}
	if err := writeLine(w, b.String()); err != nil {
		return err
	}
	if fn.Body != "" {
		if _, err := io.WriteString(w, fn.Body); err != nil {
			return fmt.Errorf("%w: %w", errWrite, err)
		}
		if !strings.HasSuffix(fn.Body, "\n") {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return fmt.Errorf("%w: %w", errWrite, err)
			}
		}
	}
	_, err := io.WriteString(w, "endef\n")
	if err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	return nil
}

func writeConditional(w io.Writer, c *Conditional) error {
	if c == nil || len(c.Branches) == 0 {
		return fmt.Errorf("%w: empty conditional", errFile)
	}
	elseSeen := false
	for i, br := range c.Branches {
		if br == nil {
			return fmt.Errorf("%w: nil conditional branch", errFile)
		}
		var b strings.Builder
		if i > 0 {
			b.WriteString("else")
			if br.HasTest {
				b.WriteByte(' ')
			}
		}
		if br.HasTest {
			if elseSeen {
				return fmt.Errorf("%w: else follows plain else", errSyntax)
			}
			test, err := formatTest(br)
			if err != nil {
				return err
			}
			b.WriteString(test)
		} else {
			if elseSeen {
				return fmt.Errorf("%w: only one else per conditional", errSyntax)
			}
			elseSeen = true
			if i == 0 {
				return fmt.Errorf("%w: conditional starts with else", errSyntax)
			}
		}
		if err := writeLine(w, b.String()); err != nil {
			return err
		}
		for _, stmt := range br.Body {
			if err := writeStmt(w, stmt); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "endif\n")
	if err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	return nil
}

func formatTest(br *Branch) (string, error) {
	switch br.Kind {
	case KindIfDef:
		if br.Left == "" {
			return "", fmt.Errorf("%w: empty ifdef", errSyntax)
		}
		return "ifdef " + br.Left, nil
	case KindIfNdef:
		if br.Left == "" {
			return "", fmt.Errorf("%w: empty ifndef", errSyntax)
		}
		return "ifndef " + br.Left, nil
	case KindIfEq, KindIfNeq:
		kw := "ifeq"
		if br.Kind == KindIfNeq {
			kw = "ifneq"
		}
		if needsQuotedTest(br.Left, br.Right) {
			left, err := quoteArg(br.Left)
			if err != nil {
				return "", fmt.Errorf("%w: conditional text", errSyntax)
			}
			right, err := quoteArg(br.Right)
			if err != nil {
				return "", fmt.Errorf("%w: conditional text", errSyntax)
			}
			return kw + " " + left + " " + right, nil
		}
		return kw + " (" + br.Left + "," + br.Right + ")", nil
	default:
		return "", fmt.Errorf("%w: conditional kind", errSyntax)
	}
}

func writeInclude(w io.Writer, in *Include) error {
	if in == nil || len(in.Paths) == 0 {
		return fmt.Errorf("%w: empty include", errFile)
	}
	kw := "include"
	if in.Optional {
		kw = "-include"
	}
	return writeLine(w, kw+" "+strings.Join(in.Paths, " "))
}

func writeExport(w io.Writer, ex *Export) error {
	if ex == nil {
		return fmt.Errorf("%w: nil export", errFile)
	}
	kw := "export"
	if ex.Drop {
		kw = "unexport"
	}
	if ex.All {
		return writeLine(w, kw)
	}
	if len(ex.Names) == 0 {
		return fmt.Errorf("%w: empty export", errFile)
	}
	return writeLine(w, kw+" "+strings.Join(ex.Names, " "))
}

func writeVPath(w io.Writer, v *VPath) error {
	if v == nil {
		return fmt.Errorf("%w: nil vpath", errFile)
	}
	if v.ClearAll {
		return writeLine(w, "vpath")
	}
	if v.Pattern == "" {
		return fmt.Errorf("%w: empty vpath", errFile)
	}
	line := "vpath " + v.Pattern
	if v.Dirs != "" {
		line += " " + v.Dirs
	}
	return writeLine(w, line)
}

func writeUndefine(w io.Writer, u *Undefine) error {
	if u == nil || len(u.Names) == 0 {
		return fmt.Errorf("%w: empty undefine", errFile)
	}
	return writeLine(w, "undefine "+strings.Join(u.Names, " "))
}

func writeMods(b *strings.Builder, override, export, private bool) {
	if override {
		b.WriteString("override ")
	}
	if export {
		b.WriteString("export ")
	}
	if private {
		b.WriteString("private ")
	}
}

func writeLine(w io.Writer, line string) error {
	_, err := io.WriteString(w, escapeLine(line)+"\n")
	if err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}
	return nil
}
