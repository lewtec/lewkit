package ninja

import (
	"fmt"
	"io"
)

// Stmt is one ninja statement, in source order.
type Stmt interface{ stmt() }

// Piece is literal text or one variable reference.
// Text and Var are mutually exclusive. Nothing in a piece is expanded.
type Piece struct {
	Text string
	Var  string
}

// Text returns a literal piece.
func Text(s string) Piece { return Piece{Text: s} }

// Ref returns a variable reference.
func Ref(name string) Piece { return Piece{Var: name} }

// Variable is one top-level binding.
// Value is nil when the binding is empty.
type Variable struct {
	Name  string
	Value []Piece
}

func (*Variable) stmt() {}

// Binding is one indented key on a rule, pool, or build edge.
type Binding struct {
	Name  string
	Value []Piece
}

// Rule is a ninja rule and its bindings.
// Bindings are kept as written. command is not required.
type Rule struct {
	Name     string
	Bindings []Binding
}

func (*Rule) stmt() {}

// Edge is one build statement.
// Outputs and Inputs are explicit. ImplicitDeps, OrderOnlyDeps, and
// Validations follow the |, ||, and |@ lists. The rule name is not
// looked up, so phony need not be declared.
type Edge struct {
	Outputs         [][]Piece
	ImplicitOutputs [][]Piece
	Rule            string
	Inputs          [][]Piece
	ImplicitDeps    [][]Piece
	OrderOnlyDeps   [][]Piece
	Validations     [][]Piece
	Bindings        []Binding
}

func (*Edge) stmt() {}

// Default is a default statement.
type Default struct {
	Targets [][]Piece
}

func (*Default) stmt() {}

// Pool is a pool and its bindings.
// depth is not required.
type Pool struct {
	Name     string
	Bindings []Binding
}

func (*Pool) stmt() {}

// Include is an include or subninja path.
// The path is not loaded.
type Include struct {
	Path     []Piece
	Subninja bool
}

func (*Include) stmt() {}

// File is one ninja file.
type File struct {
	Stmts []Stmt
}

// New returns an empty file.
func New() *File { return &File{} }

// Take reads a ninja file.
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil reader", errRead)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRead, err)
	}
	return parse(string(raw))
}

// Make writes the file.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return fmt.Errorf("%w: nil file", errFile)
	}
	if w == nil {
		return fmt.Errorf("%w: nil writer", errWrite)
	}
	return writeFile(w, f)
}

// Variable appends a binding and returns it.
func (f *File) Variable(name string) *Variable {
	if f == nil {
		return nil
	}
	v := &Variable{Name: name}
	f.Stmts = append(f.Stmts, v)
	return v
}

// Set stores the unexpanded value.
// Adjacent text pieces are joined, matching parse.
func (v *Variable) Set(pieces ...Piece) *Variable {
	if v == nil {
		return nil
	}
	v.Value = merge(pieces)
	return v
}

// Rule appends a rule and returns it.
func (f *File) Rule(name string) *Rule {
	if f == nil {
		return nil
	}
	r := &Rule{Name: name}
	f.Stmts = append(f.Stmts, r)
	return r
}

// Bind appends a binding.
func (r *Rule) Bind(name string, pieces ...Piece) *Rule {
	if r == nil {
		return nil
	}
	r.Bindings = append(r.Bindings, Binding{Name: name, Value: merge(pieces)})
	return r
}

// Build appends an edge for rule and returns it.
func (f *File) Build(rule string) *Edge {
	if f == nil {
		return nil
	}
	e := &Edge{Rule: rule}
	f.Stmts = append(f.Stmts, e)
	return e
}

// Output appends one explicit output path.
func (e *Edge) Output(pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.Outputs = append(e.Outputs, merge(pieces))
	return e
}

// ImplicitOutput appends one implicit output path.
func (e *Edge) ImplicitOutput(pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.ImplicitOutputs = append(e.ImplicitOutputs, merge(pieces))
	return e
}

// Input appends one explicit input path.
func (e *Edge) Input(pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.Inputs = append(e.Inputs, merge(pieces))
	return e
}

// Implicit appends one implicit input path.
func (e *Edge) Implicit(pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.ImplicitDeps = append(e.ImplicitDeps, merge(pieces))
	return e
}

// OrderOnly appends one order-only input path.
func (e *Edge) OrderOnly(pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.OrderOnlyDeps = append(e.OrderOnlyDeps, merge(pieces))
	return e
}

// Validation appends one validation path.
func (e *Edge) Validation(pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.Validations = append(e.Validations, merge(pieces))
	return e
}

// Bind appends an edge binding.
func (e *Edge) Bind(name string, pieces ...Piece) *Edge {
	if e == nil {
		return nil
	}
	e.Bindings = append(e.Bindings, Binding{Name: name, Value: merge(pieces)})
	return e
}

// Default appends a default statement and returns it.
func (f *File) Default(targets ...[]Piece) *Default {
	if f == nil {
		return nil
	}
	d := &Default{}
	for _, t := range targets {
		d.Targets = append(d.Targets, merge(t))
	}
	f.Stmts = append(f.Stmts, d)
	return d
}

// Pool appends a pool and returns it.
func (f *File) Pool(name string) *Pool {
	if f == nil {
		return nil
	}
	p := &Pool{Name: name}
	f.Stmts = append(f.Stmts, p)
	return p
}

// Bind appends a pool binding.
func (p *Pool) Bind(name string, pieces ...Piece) *Pool {
	if p == nil {
		return nil
	}
	p.Bindings = append(p.Bindings, Binding{Name: name, Value: merge(pieces)})
	return p
}

// Include appends an include and returns it.
func (f *File) Include(pieces ...Piece) *Include {
	if f == nil {
		return nil
	}
	inc := &Include{Path: merge(pieces)}
	f.Stmts = append(f.Stmts, inc)
	return inc
}

// Subninja appends a subninja and returns it.
func (f *File) Subninja(pieces ...Piece) *Include {
	if f == nil {
		return nil
	}
	inc := &Include{Path: merge(pieces), Subninja: true}
	f.Stmts = append(f.Stmts, inc)
	return inc
}

func merge(pieces []Piece) []Piece {
	if len(pieces) == 0 {
		return nil
	}
	var out []Piece
	for _, p := range pieces {
		switch {
		case p.Text != "" && p.Var != "":
			out = append(out, p)
		case p.Var != "":
			out = append(out, Piece{Var: p.Var})
		case p.Text != "":
			addText(&out, p.Text)
		default:
			out = append(out, Piece{})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func addText(ps *[]Piece, s string) {
	if s == "" {
		return
	}
	if n := len(*ps); n > 0 && (*ps)[n-1].Var == "" && (*ps)[n-1].Text != "" {
		(*ps)[n-1].Text += s
		return
	}
	*ps = append(*ps, Piece{Text: s})
}

func addVar(ps *[]Piece, name string) {
	*ps = append(*ps, Piece{Var: name})
}
