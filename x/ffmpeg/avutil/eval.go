package avutil

import (
	"fmt"
	"math"
	"strconv"
	"unicode"
)

// Expr is a parsed av_expr expression.
type Expr struct {
	text string
	eval func(vars map[string]float64) float64
}

// ParseExpr compiles text. Names are the variables the expression may read.
func ParseExpr(text string, names ...string) (*Expr, error) {
	p := &parser{s: text, i: 0, names: map[string]struct{}{}}
	for _, name := range names {
		p.names[name] = struct{}{}
	}
	fn, err := p.parseSeq()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.i != len(p.s) {
		return nil, fmt.Errorf("%w: trailing %q", ErrOption, p.s[p.i:])
	}
	return &Expr{text: text, eval: fn}, nil
}

// Eval runs the expression. Unknown variables are zero.
func (e *Expr) Eval(vars map[string]float64) float64 {
	if e == nil || e.eval == nil {
		return math.NaN()
	}
	return e.eval(vars)
}

// Text is the source.
func (e *Expr) Text() string {
	if e == nil {
		return ""
	}
	return e.text
}

type parser struct {
	s     string
	i     int
	names map[string]struct{}
}

func (p *parser) skip() {
	for p.i < len(p.s) && unicode.IsSpace(rune(p.s[p.i])) {
		p.i++
	}
}

func (p *parser) parseSeq() (func(map[string]float64) float64, error) {
	left, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == ';' {
		p.i++
		right, err := p.parseSeq()
		if err != nil {
			return nil, err
		}
		return func(vars map[string]float64) float64 {
			left(vars)
			return right(vars)
		}, nil
	}
	return left, nil
}

func (p *parser) parseAdd() (func(map[string]float64) float64, error) {
	left, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for {
		p.skip()
		if p.i >= len(p.s) {
			return left, nil
		}
		op := p.s[p.i]
		if op != '+' && op != '-' {
			return left, nil
		}
		p.i++
		right, err := p.parseMul()
		if err != nil {
			return nil, err
		}
		lhs := left
		left = func(vars map[string]float64) float64 {
			if op == '+' {
				return lhs(vars) + right(vars)
			}
			return lhs(vars) - right(vars)
		}
	}
}

func (p *parser) parseMul() (func(map[string]float64) float64, error) {
	left, err := p.parsePow()
	if err != nil {
		return nil, err
	}
	for {
		p.skip()
		if p.i >= len(p.s) {
			return left, nil
		}
		op := p.s[p.i]
		if op != '*' && op != '/' {
			return left, nil
		}
		p.i++
		right, err := p.parsePow()
		if err != nil {
			return nil, err
		}
		lhs := left
		left = func(vars map[string]float64) float64 {
			if op == '*' {
				return lhs(vars) * right(vars)
			}
			den := right(vars)
			if den == 0 {
				return math.NaN()
			}
			return lhs(vars) / den
		}
	}
}

func (p *parser) parsePow() (func(map[string]float64) float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == '^' {
		p.i++
		right, err := p.parsePow()
		if err != nil {
			return nil, err
		}
		return func(vars map[string]float64) float64 {
			return math.Pow(left(vars), right(vars))
		}, nil
	}
	return left, nil
}

func (p *parser) parseUnary() (func(map[string]float64) float64, error) {
	p.skip()
	if p.i >= len(p.s) {
		return nil, fmt.Errorf("%w: truncated expression", ErrOption)
	}
	switch p.s[p.i] {
	case '+':
		p.i++
		return p.parseUnary()
	case '-':
		p.i++
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return func(vars map[string]float64) float64 { return -inner(vars) }, nil
	default:
		return p.parsePrimary()
	}
}

func (p *parser) parsePrimary() (func(map[string]float64) float64, error) {
	p.skip()
	if p.i >= len(p.s) {
		return nil, fmt.Errorf("%w: truncated expression", ErrOption)
	}
	if p.s[p.i] == '(' {
		p.i++
		inner, err := p.parseSeq()
		if err != nil {
			return nil, err
		}
		p.skip()
		if p.i >= len(p.s) || p.s[p.i] != ')' {
			return nil, fmt.Errorf("%w: missing )", ErrOption)
		}
		p.i++
		return inner, nil
	}
	if isIdent(p.s[p.i]) {
		start := p.i
		p.i++
		for p.i < len(p.s) && (isIdent(p.s[p.i]) || unicode.IsDigit(rune(p.s[p.i]))) {
			p.i++
		}
		name := p.s[start:p.i]
		p.skip()
		if p.i < len(p.s) && p.s[p.i] == '(' {
			return p.parseCall(name)
		}
		switch name {
		case "PI":
			return func(map[string]float64) float64 { return math.Pi }, nil
		case "E":
			return func(map[string]float64) float64 { return math.E }, nil
		}
		return func(vars map[string]float64) float64 {
			if vars == nil {
				return 0
			}
			return vars[name]
		}, nil
	}
	return p.parseNumber()
}

func (p *parser) parseCall(name string) (func(map[string]float64) float64, error) {
	p.i++
	var args []func(map[string]float64) float64
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == ')' {
		p.i++
	} else {
		for {
			arg, err := p.parseSeq()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			p.skip()
			if p.i >= len(p.s) {
				return nil, fmt.Errorf("%w: missing )", ErrOption)
			}
			if p.s[p.i] == ',' {
				p.i++
				continue
			}
			if p.s[p.i] == ')' {
				p.i++
				break
			}
			return nil, fmt.Errorf("%w: call %s", ErrOption, name)
		}
	}
	fn, ok := exprFuncs[name]
	if !ok {
		return nil, fmt.Errorf("%w: function %s", ErrOption, name)
	}
	return func(vars map[string]float64) float64 {
		got := make([]float64, len(args))
		for i := range args {
			got[i] = args[i](vars)
		}
		return fn(got)
	}, nil
}

func (p *parser) parseNumber() (func(map[string]float64) float64, error) {
	start := p.i
	if p.i+1 < len(p.s) && p.s[p.i] == '0' && (p.s[p.i+1] == 'x' || p.s[p.i+1] == 'X') {
		p.i += 2
		for p.i < len(p.s) && unicode.Is(unicode.ASCII_Hex_Digit, rune(p.s[p.i])) {
			p.i++
		}
		value, err := parseHex(p.s[start:p.i])
		if err != nil {
			return nil, err
		}
		return func(map[string]float64) float64 { return value }, nil
	}
	for p.i < len(p.s) && (unicode.IsDigit(rune(p.s[p.i])) || p.s[p.i] == '.') {
		p.i++
	}
	if p.i < len(p.s) && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
		p.i++
		if p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
			p.i++
		}
		for p.i < len(p.s) && unicode.IsDigit(rune(p.s[p.i])) {
			p.i++
		}
	}
	if start == p.i {
		return nil, fmt.Errorf("%w: expression at %q", ErrOption, p.s[start:])
	}
	value, err := mathParse(p.s[start:p.i])
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOption, p.s[start:p.i])
	}
	return func(map[string]float64) float64 { return value }, nil
}

func isIdent(b byte) bool {
	return b == '_' || unicode.IsLetter(rune(b))
}

func parseHex(text string) (float64, error) {
	var n uint64
	for _, r := range text[2:] {
		n <<= 4
		switch {
		case r >= '0' && r <= '9':
			n += uint64(r - '0')
		case r >= 'a' && r <= 'f':
			n += uint64(r-'a') + 10
		case r >= 'A' && r <= 'F':
			n += uint64(r-'A') + 10
		default:
			return 0, fmt.Errorf("%w: %s", ErrOption, text)
		}
	}
	return float64(n), nil
}

func mathParse(text string) (float64, error) {
	return strconv.ParseFloat(text, 64)
}

func exprArg(args []float64, i int) float64 {
	if i >= len(args) {
		return math.NaN()
	}
	return args[i]
}

var exprFuncs = map[string]func([]float64) float64{
	"abs":     func(a []float64) float64 { return math.Abs(exprArg(a, 0)) },
	"min":     func(a []float64) float64 { return math.Min(exprArg(a, 0), exprArg(a, 1)) },
	"max":     func(a []float64) float64 { return math.Max(exprArg(a, 0), exprArg(a, 1)) },
	"clip":    func(a []float64) float64 { return math.Min(math.Max(exprArg(a, 0), exprArg(a, 1)), exprArg(a, 2)) },
	"sin":     func(a []float64) float64 { return math.Sin(exprArg(a, 0)) },
	"cos":     func(a []float64) float64 { return math.Cos(exprArg(a, 0)) },
	"tan":     func(a []float64) float64 { return math.Tan(exprArg(a, 0)) },
	"atan":    func(a []float64) float64 { return math.Atan(exprArg(a, 0)) },
	"atan2":   func(a []float64) float64 { return math.Atan2(exprArg(a, 0), exprArg(a, 1)) },
	"asin":    func(a []float64) float64 { return math.Asin(exprArg(a, 0)) },
	"acos":    func(a []float64) float64 { return math.Acos(exprArg(a, 0)) },
	"exp":     func(a []float64) float64 { return math.Exp(exprArg(a, 0)) },
	"log":     func(a []float64) float64 { return math.Log(exprArg(a, 0)) },
	"sqrt":    func(a []float64) float64 { return math.Sqrt(exprArg(a, 0)) },
	"hypot":   func(a []float64) float64 { return math.Hypot(exprArg(a, 0), exprArg(a, 1)) },
	"floor":   func(a []float64) float64 { return math.Floor(exprArg(a, 0)) },
	"ceil":    func(a []float64) float64 { return math.Ceil(exprArg(a, 0)) },
	"trunc":   func(a []float64) float64 { return math.Trunc(exprArg(a, 0)) },
	"round":   func(a []float64) float64 { return math.Round(exprArg(a, 0)) },
	"pow":     func(a []float64) float64 { return math.Pow(exprArg(a, 0), exprArg(a, 1)) },
	"mod":     func(a []float64) float64 { return math.Mod(exprArg(a, 0), exprArg(a, 1)) },
	"eq":      func(a []float64) float64 { return boolFloat(exprArg(a, 0) == exprArg(a, 1)) },
	"gte":     func(a []float64) float64 { return boolFloat(exprArg(a, 0) >= exprArg(a, 1)) },
	"gt":      func(a []float64) float64 { return boolFloat(exprArg(a, 0) > exprArg(a, 1)) },
	"lte":     func(a []float64) float64 { return boolFloat(exprArg(a, 0) <= exprArg(a, 1)) },
	"lt":      func(a []float64) float64 { return boolFloat(exprArg(a, 0) < exprArg(a, 1)) },
	"not":     func(a []float64) float64 { return boolFloat(exprArg(a, 0) == 0) },
	"between": func(a []float64) float64 {
		v := exprArg(a, 0)
		return boolFloat(v >= exprArg(a, 1) && v <= exprArg(a, 2))
	},
	"if": func(a []float64) float64 {
		if exprArg(a, 0) != 0 {
			return exprArg(a, 1)
		}
		return exprArg(a, 2)
	},
	"ifnot": func(a []float64) float64 {
		if exprArg(a, 0) == 0 {
			return exprArg(a, 1)
		}
		return exprArg(a, 2)
	},
	"lerp": func(a []float64) float64 {
		x, y, t := exprArg(a, 0), exprArg(a, 1), exprArg(a, 2)
		return x + (y-x)*t
	},
	"squish": func(a []float64) float64 { return 1 / (1 + math.Exp(4*exprArg(a, 0))) },
	"gauss": func(a []float64) float64 {
		x := exprArg(a, 0)
		return math.Exp(-x * x / 2)
	},
	"isnan": func(a []float64) float64 { return boolFloat(math.IsNaN(exprArg(a, 0))) },
	"isinf": func(a []float64) float64 { return boolFloat(math.IsInf(exprArg(a, 0), 0)) },
	"sinh":  func(a []float64) float64 { return math.Sinh(exprArg(a, 0)) },
	"cosh":  func(a []float64) float64 { return math.Cosh(exprArg(a, 0)) },
	"tanh":  func(a []float64) float64 { return math.Tanh(exprArg(a, 0)) },
}

func boolFloat(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}
