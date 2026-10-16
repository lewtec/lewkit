package golang

// Code is an expression or a statement placed in a body.
type Code interface {
	code()
}

// Stmt is one statement. The kind selects the fields in use.
//
//	Expr    Expr
//	Return  Exprs
//	Assign  LHS, RHS, Op, Define
//	If      Init, Cond, Body, ElseBranch
//	For     Init, Cond, Post, Body
//	Range   Key, Value, X, Body, Define
//	Block   Body
//	Decl    Decl
//	Inc     X
//	Dec     X
//	Defer   Expr
//	Go      Expr
//	Branch  Branch, Label
//	Send        X, Y
//	Label       Label, Stmt
//	Switch      Init, Cond, Body
//	TypeSwitch  Init, Stmt, Body
//	Select      Body
//	Case        Exprs, Body. Nil Exprs is default.
//	Comm        Stmt, Body. Nil Stmt is default.
type Stmt struct {
	Kind       StmtKind
	Expr       *Expr
	Exprs      []*Expr
	LHS        []*Expr
	RHS        []*Expr
	Op         Op
	Define     bool
	Init       *Stmt
	Cond       *Expr
	Post       *Stmt
	Body       []*Stmt
	ElseBranch *Stmt
	Key        *Expr
	Value      *Expr
	X          *Expr
	Y          *Expr
	Decl       Decl
	Branch     Branch
	Label      string
	Stmt       *Stmt
}

func (*Stmt) code() {}

func statements(lines []Code) []*Stmt {
	out := make([]*Stmt, 0, len(lines))
	for _, line := range lines {
		if line == nil {
			continue
		}
		switch n := line.(type) {
		case *Expr:
			out = append(out, &Stmt{Kind: StmtExpr, Expr: n})
		case *Stmt:
			out = append(out, n)
		}
	}
	return out
}

// Return returns a return statement.
func Return(vals ...*Expr) *Stmt {
	if vals == nil {
		vals = []*Expr{}
	}
	return &Stmt{Kind: StmtReturn, Exprs: vals}
}

// Define returns a short variable declaration.
func Define(lhs []*Expr, rhs ...*Expr) *Stmt {
	return &Stmt{Kind: StmtAssign, Define: true, LHS: lhs, RHS: rhs}
}

// Assign returns an assignment. op is zero for "=".
func Assign(op Op, lhs []*Expr, rhs ...*Expr) *Stmt {
	return &Stmt{Kind: StmtAssign, Op: op, LHS: lhs, RHS: rhs}
}

// If returns an if statement. [Stmt.Else] and [Stmt.WithInit] extend it.
func If(cond *Expr, body ...Code) *Stmt {
	return &Stmt{Kind: StmtIf, Cond: cond, Body: statements(body)}
}

// Else sets the else branch to a block.
func (s *Stmt) Else(body ...Code) *Stmt {
	if s != nil {
		s.ElseBranch = &Stmt{Kind: StmtBlock, Body: statements(body)}
	}
	return s
}

// ElseIf sets the else branch to another if.
func (s *Stmt) ElseIf(cond *Expr, body ...Code) *Stmt {
	if s != nil {
		s.ElseBranch = If(cond, body...)
	}
	return s
}

// WithInit sets the simple statement that opens an if or for.
func (s *Stmt) WithInit(init *Stmt) *Stmt {
	if s != nil {
		s.Init = init
	}
	return s
}

// For returns a for statement. A nil condition is an unconditional loop.
func For(cond *Expr, body ...Code) *Stmt {
	return &Stmt{Kind: StmtFor, Cond: cond, Body: statements(body)}
}

// WithPost sets the post statement of a for loop.
func (s *Stmt) WithPost(post *Stmt) *Stmt {
	if s != nil {
		s.Post = post
	}
	return s
}

// Range returns a range loop. [Stmt.WithKey] names the iteration variables.
func Range(x *Expr, body ...Code) *Stmt {
	return &Stmt{Kind: StmtRange, X: x, Body: statements(body)}
}

// WithKey sets the range variables. define selects ":=" over "=".
func (s *Stmt) WithKey(define bool, key, value *Expr) *Stmt {
	if s != nil {
		s.Define = define
		s.Key = key
		s.Value = value
	}
	return s
}

// Block returns a nested block.
func Block(body ...Code) *Stmt {
	return &Stmt{Kind: StmtBlock, Body: statements(body)}
}

// Inc returns an increment statement.
func Inc(x *Expr) *Stmt {
	return &Stmt{Kind: StmtInc, X: x}
}

// Dec returns a decrement statement.
func Dec(x *Expr) *Stmt {
	return &Stmt{Kind: StmtDec, X: x}
}

// Defer returns a defer statement. expr is the call.
func Defer(expr *Expr) *Stmt {
	return &Stmt{Kind: StmtDefer, Expr: expr}
}

// Go returns a go statement. expr is the call.
func Go(expr *Expr) *Stmt {
	return &Stmt{Kind: StmtGo, Expr: expr}
}

// Break returns a break statement. label may be empty.
func Break(label string) *Stmt {
	return &Stmt{Kind: StmtBranch, Branch: BranchBreak, Label: label}
}

// Continue returns a continue statement. label may be empty.
func Continue(label string) *Stmt {
	return &Stmt{Kind: StmtBranch, Branch: BranchContinue, Label: label}
}

// Goto returns a goto statement.
func Goto(label string) *Stmt {
	return &Stmt{Kind: StmtBranch, Branch: BranchGoto, Label: label}
}

// Fallthrough returns a fallthrough statement.
func Fallthrough() *Stmt {
	return &Stmt{Kind: StmtBranch, Branch: BranchFallthrough}
}

// Send returns ch <- value.
func Send(ch, value *Expr) *Stmt {
	return &Stmt{Kind: StmtSend, X: ch, Y: value}
}

// Label returns a labeled statement.
func Label(label string, stmt *Stmt) *Stmt {
	return &Stmt{Kind: StmtLabel, Label: label, Stmt: stmt}
}
