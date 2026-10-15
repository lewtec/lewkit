package sql

import "fmt"

// Expr is one SQL expression.
// Name is an identifier, a function, or an operator.
// Text is a string, a number, or a placeholder.
// Args are call arguments or the operands of an operator.
type Expr struct {
	Kind ExprKind
	Name string
	Text string
	Args []Expr
}

// Id is an identifier. A qualified name keeps the dots.
func Id(name string) Expr {
	return Expr{Kind: ExprKindIdent, Name: name}
}

// Lit is a string literal.
func Lit(text string) Expr {
	return Expr{Kind: ExprKindString, Text: text}
}

// Num is a numeric literal spelled as text.
func Num(text string) Expr {
	return Expr{Kind: ExprKindNumber, Text: text}
}

// Arg is the ? placeholder.
func Arg() Expr {
	return Expr{Kind: ExprKindPlaceholder, Text: "?"}
}

// ArgN is the $n placeholder. n is the 1-based index.
func ArgN(n int) Expr {
	return Expr{Kind: ExprKindPlaceholder, Text: fmt.Sprintf("$%d", n)}
}

// Star is *.
func Star() Expr {
	return Expr{Kind: ExprKindStar}
}

// Call is a function call. The name may be qualified, as in sqlc.arg.
func Call(name string, args ...Expr) Expr {
	return Expr{Kind: ExprKindCall, Name: name, Args: append([]Expr(nil), args...)}
}

// Eq is left = right.
func Eq(left, right Expr) Expr { return binary("=", left, right) }

// NotEq is left <> right.
func NotEq(left, right Expr) Expr { return binary("<>", left, right) }

// Less is left < right.
func Less(left, right Expr) Expr { return binary("<", left, right) }

// Greater is left > right.
func Greater(left, right Expr) Expr { return binary(">", left, right) }

// Like is left LIKE right.
func Like(left, right Expr) Expr { return binary("LIKE", left, right) }

// IsNull is expr IS NULL.
func IsNull(expr Expr) Expr { return binary("IS", expr, Id("NULL")) }

// IsNotNull is expr IS NOT NULL.
func IsNotNull(expr Expr) Expr { return binary("IS NOT", expr, Id("NULL")) }

// Not is NOT expr.
func Not(expr Expr) Expr { return unary("NOT", expr) }

// And is a left-associated AND chain.
func And(items ...Expr) Expr { return fold("AND", items) }

// Or is a left-associated OR chain.
func Or(items ...Expr) Expr { return fold("OR", items) }

func unary(op string, expr Expr) Expr {
	return Expr{Kind: ExprKindUnary, Name: op, Args: []Expr{expr}}
}

func binary(op string, left, right Expr) Expr {
	return Expr{Kind: ExprKindBinary, Name: op, Args: []Expr{left, right}}
}

func fold(op string, items []Expr) Expr {
	if len(items) == 0 {
		return Expr{}
	}
	out := items[0]
	for _, item := range items[1:] {
		out = binary(op, out, item)
	}
	return out
}
