package makefile

import (
	"fmt"
	"io"
)

// Stmt is one makefile statement, in source order.
// Conditionals are stored, not evaluated.
type Stmt interface{ stmt() }

// Variable is one assignment.
// Targets selects a target-specific variable.
// Value is the read-time text: continuations are already joined
// and comments are already removed. Nothing in Value is expanded.
type Variable struct {
	Name     string
	Op       Op
	Value    string
	Targets  []string
	Override bool
	Export   bool
	Private  bool
}

func (*Variable) stmt() {}

// Rule is targets, prerequisites, and a recipe.
// Pattern is set for a static pattern rule.
// A rule with no targets is a recipe that stands alone inside a conditional.
type Rule struct {
	Targets       []string
	Pattern       string
	Prerequisites []string
	OrderOnly     []string
	Recipe        []string
	DoubleColon   bool
	Grouped       bool
}

func (*Rule) stmt() {}

// Function is a define/endef block.
// A zero Op prints define with no operator.
// Body keeps newlines and comments. Continuations are already joined.
type Function struct {
	Name     string
	Op       Op
	Body     string
	Override bool
	Export   bool
	Private  bool
}

func (*Function) stmt() {}

// Branch is one arm of a conditional.
// HasTest is false on a plain else.
type Branch struct {
	Kind    Kind
	HasTest bool
	Left    string
	Right   string
	Body    []Stmt
}

func (b *Branch) add(s Stmt) {
	b.Body = append(b.Body, s)
}

func (b *Branch) last() Stmt {
	if b == nil || len(b.Body) == 0 {
		return nil
	}
	return b.Body[len(b.Body)-1]
}

func (*Branch) top() bool { return false }

// Conditional is one if/else/endif chain.
// An else if is another branch, closed by the same endif.
type Conditional struct {
	Branches []*Branch
}

func (*Conditional) stmt() {}

// Include is an include directive.
// Optional is -include. sinclude is read as optional.
type Include struct {
	Paths    []string
	Optional bool
}

func (*Include) stmt() {}

// Export is export or unexport outside an assignment.
// All with Drop false exports every variable.
// All with Drop true unexports every variable.
type Export struct {
	Names []string
	All   bool
	Drop  bool
}

func (*Export) stmt() {}

// VPath is a vpath directive.
// ClearAll is a bare vpath.
// Dirs is the search path with its colons kept together.
type VPath struct {
	ClearAll bool
	Pattern  string
	Dirs     string
}

func (*VPath) stmt() {}

// Undefine is an undefine directive.
type Undefine struct {
	Names []string
}

func (*Undefine) stmt() {}

// File is one Makefile.
type File struct {
	Stmts []Stmt
}

func (f *File) add(s Stmt) {
	f.Stmts = append(f.Stmts, s)
}

func (f *File) last() Stmt {
	if f == nil || len(f.Stmts) == 0 {
		return nil
	}
	return f.Stmts[len(f.Stmts)-1]
}

func (*File) top() bool { return true }

// New returns an empty file.
func New() *File { return &File{} }

// Variable appends a recursive assignment and returns it.
func (f *File) Variable(name string) *Variable {
	if f == nil {
		return nil
	}
	return addVariable(&f.Stmts, name)
}

// Rule appends a rule and returns it.
func (f *File) Rule(targets ...string) *Rule {
	if f == nil {
		return nil
	}
	return addRule(&f.Stmts, targets...)
}

// Function appends a define block and returns it.
func (f *File) Function(name string) *Function {
	if f == nil {
		return nil
	}
	return addFunction(&f.Stmts, name)
}

// Include appends an include directive and returns it.
func (f *File) Include(paths ...string) *Include {
	if f == nil {
		return nil
	}
	return addInclude(&f.Stmts, paths...)
}

// Export appends an export directive and returns it.
// With no names, every variable is exported.
func (f *File) Export(names ...string) *Export {
	if f == nil {
		return nil
	}
	return addExport(&f.Stmts, false, names...)
}

// Unexport appends an unexport directive and returns it.
// With no names, every variable is unexported.
func (f *File) Unexport(names ...string) *Export {
	if f == nil {
		return nil
	}
	return addExport(&f.Stmts, true, names...)
}

// VPath appends a selective vpath and returns it.
func (f *File) VPath(pattern, dirs string) *VPath {
	if f == nil {
		return nil
	}
	return addVPath(&f.Stmts, pattern, dirs)
}

// VPathClear appends a bare vpath and returns it.
func (f *File) VPathClear() *VPath {
	if f == nil {
		return nil
	}
	v := &VPath{ClearAll: true}
	f.Stmts = append(f.Stmts, v)
	return v
}

// Undefine appends an undefine directive and returns it.
func (f *File) Undefine(names ...string) *Undefine {
	if f == nil {
		return nil
	}
	return addUndefine(&f.Stmts, names...)
}

// Conditional appends an empty conditional and returns it.
func (f *File) Conditional() *Conditional {
	if f == nil {
		return nil
	}
	return addConditional(&f.Stmts)
}

// IfEq appends a one-arm ifeq and returns that arm.
func (f *File) IfEq(left, right string) *Branch {
	if f == nil {
		return nil
	}
	return f.Conditional().IfEq(left, right)
}

// IfNeq appends a one-arm ifneq and returns that arm.
func (f *File) IfNeq(left, right string) *Branch {
	if f == nil {
		return nil
	}
	return f.Conditional().IfNeq(left, right)
}

// IfDef appends a one-arm ifdef and returns that arm.
func (f *File) IfDef(name string) *Branch {
	if f == nil {
		return nil
	}
	return f.Conditional().IfDef(name)
}

// IfNdef appends a one-arm ifndef and returns that arm.
func (f *File) IfNdef(name string) *Branch {
	if f == nil {
		return nil
	}
	return f.Conditional().IfNdef(name)
}

// Set stores the unexpanded value.
func (v *Variable) Set(value string) *Variable {
	if v == nil {
		return nil
	}
	v.Value = value
	return v
}

// Simple selects :=.
func (v *Variable) Simple() *Variable { return v.setOp(OpSimple) }

// Immediate selects ::=.
func (v *Variable) Immediate() *Variable { return v.setOp(OpImmediate) }

// Shell selects !=.
// The text is kept; it is not run.
func (v *Variable) Shell() *Variable { return v.setOp(OpShell) }

// Append selects +=.
func (v *Variable) Append() *Variable { return v.setOp(OpAppend) }

// IfUnset selects ?=.
func (v *Variable) IfUnset() *Variable { return v.setOp(OpConditional) }

// Exported marks the assignment exported.
func (v *Variable) Exported() *Variable {
	if v != nil {
		v.Export = true
	}
	return v
}

// Overridden marks the assignment override.
func (v *Variable) Overridden() *Variable {
	if v != nil {
		v.Override = true
	}
	return v
}

// Privately marks the assignment private.
func (v *Variable) Privately() *Variable {
	if v != nil {
		v.Private = true
	}
	return v
}

// On makes the assignment target-specific.
func (v *Variable) On(targets ...string) *Variable {
	if v == nil {
		return nil
	}
	v.Targets = append(v.Targets, targets...)
	return v
}

func (v *Variable) setOp(op Op) *Variable {
	if v != nil {
		v.Op = op
	}
	return v
}

// Static sets the static-pattern stem.
func (r *Rule) Static(pattern string) *Rule {
	if r == nil {
		return nil
	}
	r.Pattern = pattern
	return r
}

// Double selects a double-colon rule.
func (r *Rule) Double() *Rule {
	if r != nil {
		r.DoubleColon = true
	}
	return r
}

// Group selects a grouped &: rule.
func (r *Rule) Group() *Rule {
	if r != nil {
		r.Grouped = true
	}
	return r
}

// Prereq appends normal prerequisites.
func (r *Rule) Prereq(paths ...string) *Rule {
	if r == nil {
		return nil
	}
	r.Prerequisites = append(r.Prerequisites, paths...)
	return r
}

// Order appends order-only prerequisites.
func (r *Rule) Order(paths ...string) *Rule {
	if r == nil {
		return nil
	}
	r.OrderOnly = append(r.OrderOnly, paths...)
	return r
}

// Commands appends recipe lines.
// Each line is the text after the leading tab.
func (r *Rule) Commands(lines ...string) *Rule {
	if r == nil {
		return nil
	}
	r.Recipe = append(r.Recipe, lines...)
	return r
}

// Set stores the define body.
func (fn *Function) Set(body string) *Function {
	if fn == nil {
		return nil
	}
	fn.Body = body
	return fn
}

// Simple selects := on the define.
func (fn *Function) Simple() *Function { return fn.setOp(OpSimple) }

// Immediate selects ::= on the define.
func (fn *Function) Immediate() *Function { return fn.setOp(OpImmediate) }

// Shell selects != on the define.
func (fn *Function) Shell() *Function { return fn.setOp(OpShell) }

// Append selects += on the define.
func (fn *Function) Append() *Function { return fn.setOp(OpAppend) }

// IfUnset selects ?= on the define.
func (fn *Function) IfUnset() *Function { return fn.setOp(OpConditional) }

// Exported marks the function exported.
func (fn *Function) Exported() *Function {
	if fn != nil {
		fn.Export = true
	}
	return fn
}

// Overridden marks the function override.
func (fn *Function) Overridden() *Function {
	if fn != nil {
		fn.Override = true
	}
	return fn
}

// Privately marks the function private.
func (fn *Function) Privately() *Function {
	if fn != nil {
		fn.Private = true
	}
	return fn
}

func (fn *Function) setOp(op Op) *Function {
	if fn != nil {
		fn.Op = op
	}
	return fn
}

// Maybe marks the include optional (-include).
func (in *Include) Maybe() *Include {
	if in != nil {
		in.Optional = true
	}
	return in
}

// IfEq appends an ifeq arm and returns it.
func (c *Conditional) IfEq(left, right string) *Branch {
	return c.arm(KindIfEq, true, left, right)
}

// IfNeq appends an ifneq arm and returns it.
func (c *Conditional) IfNeq(left, right string) *Branch {
	return c.arm(KindIfNeq, true, left, right)
}

// IfDef appends an ifdef arm and returns it.
func (c *Conditional) IfDef(name string) *Branch {
	return c.arm(KindIfDef, true, name, "")
}

// IfNdef appends an ifndef arm and returns it.
func (c *Conditional) IfNdef(name string) *Branch {
	return c.arm(KindIfNdef, true, name, "")
}

// Else appends a plain else and returns it.
func (c *Conditional) Else() *Branch {
	return c.arm(0, false, "", "")
}

func (c *Conditional) arm(kind Kind, test bool, left, right string) *Branch {
	if c == nil {
		return nil
	}
	br := &Branch{Kind: kind, HasTest: test, Left: left, Right: right}
	c.Branches = append(c.Branches, br)
	return br
}

// Variable appends a recursive assignment to this arm.
func (b *Branch) Variable(name string) *Variable {
	if b == nil {
		return nil
	}
	return addVariable(&b.Body, name)
}

// Rule appends a rule to this arm.
func (b *Branch) Rule(targets ...string) *Rule {
	if b == nil {
		return nil
	}
	return addRule(&b.Body, targets...)
}

// Function appends a define block to this arm.
func (b *Branch) Function(name string) *Function {
	if b == nil {
		return nil
	}
	return addFunction(&b.Body, name)
}

// Include appends an include to this arm.
func (b *Branch) Include(paths ...string) *Include {
	if b == nil {
		return nil
	}
	return addInclude(&b.Body, paths...)
}

// Export appends an export to this arm.
func (b *Branch) Export(names ...string) *Export {
	if b == nil {
		return nil
	}
	return addExport(&b.Body, false, names...)
}

// Unexport appends an unexport to this arm.
func (b *Branch) Unexport(names ...string) *Export {
	if b == nil {
		return nil
	}
	return addExport(&b.Body, true, names...)
}

// VPath appends a selective vpath to this arm.
func (b *Branch) VPath(pattern, dirs string) *VPath {
	if b == nil {
		return nil
	}
	return addVPath(&b.Body, pattern, dirs)
}

// VPathClear appends a bare vpath to this arm.
func (b *Branch) VPathClear() *VPath {
	if b == nil {
		return nil
	}
	v := &VPath{ClearAll: true}
	b.Body = append(b.Body, v)
	return v
}

// Undefine appends an undefine to this arm.
func (b *Branch) Undefine(names ...string) *Undefine {
	if b == nil {
		return nil
	}
	return addUndefine(&b.Body, names...)
}

// Conditional appends a nested conditional to this arm.
func (b *Branch) Conditional() *Conditional {
	if b == nil {
		return nil
	}
	return addConditional(&b.Body)
}

// IfEq appends a nested ifeq and returns that arm.
func (b *Branch) IfEq(left, right string) *Branch {
	if b == nil {
		return nil
	}
	return b.Conditional().IfEq(left, right)
}

// IfNeq appends a nested ifneq and returns that arm.
func (b *Branch) IfNeq(left, right string) *Branch {
	if b == nil {
		return nil
	}
	return b.Conditional().IfNeq(left, right)
}

// IfDef appends a nested ifdef and returns that arm.
func (b *Branch) IfDef(name string) *Branch {
	if b == nil {
		return nil
	}
	return b.Conditional().IfDef(name)
}

// IfNdef appends a nested ifndef and returns that arm.
func (b *Branch) IfNdef(name string) *Branch {
	if b == nil {
		return nil
	}
	return b.Conditional().IfNdef(name)
}

func addVariable(dst *[]Stmt, name string) *Variable {
	v := &Variable{Name: name, Op: OpRecursive}
	*dst = append(*dst, v)
	return v
}

func addRule(dst *[]Stmt, targets ...string) *Rule {
	r := &Rule{Targets: append([]string(nil), targets...)}
	*dst = append(*dst, r)
	return r
}

func addFunction(dst *[]Stmt, name string) *Function {
	fn := &Function{Name: name}
	*dst = append(*dst, fn)
	return fn
}

func addInclude(dst *[]Stmt, paths ...string) *Include {
	in := &Include{Paths: append([]string(nil), paths...)}
	*dst = append(*dst, in)
	return in
}

func addExport(dst *[]Stmt, drop bool, names ...string) *Export {
	ex := &Export{Names: append([]string(nil), names...), Drop: drop}
	if len(names) == 0 {
		ex.Names = nil
		ex.All = true
	}
	*dst = append(*dst, ex)
	return ex
}

func addVPath(dst *[]Stmt, pattern, dirs string) *VPath {
	v := &VPath{Pattern: pattern, Dirs: dirs}
	*dst = append(*dst, v)
	return v
}

func addUndefine(dst *[]Stmt, names ...string) *Undefine {
	u := &Undefine{Names: append([]string(nil), names...)}
	*dst = append(*dst, u)
	return u
}

func addConditional(dst *[]Stmt) *Conditional {
	c := &Conditional{}
	*dst = append(*dst, c)
	return c
}

// Take reads a Makefile.
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

// Make writes the Makefile.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return fmt.Errorf("%w: nil file", errFile)
	}
	if w == nil {
		return fmt.Errorf("%w: nil writer", errWrite)
	}
	return writeFile(w, f)
}
