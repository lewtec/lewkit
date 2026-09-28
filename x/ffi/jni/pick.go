package jni

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// Java reflection modifiers used to choose a callable member.
const (
	modStatic = 0x0008
	modBridge = 0x0040
)

var (
	errArgument  = errors.New("unsupported java argument")
	errNoMethod  = errors.New("no method")
	errAmbiguous = errors.New("ambiguous method")
)

type kind int

const (
	kindNil kind = iota
	kindBool
	kindInt
	kindLong
	kindFloat
	kindDouble
	kindString
	kindRef
)

// value is one Go argument after classification.
type value struct {
	kind  kind
	class string
	b     bool
	i32   int32
	i64   int64
	f32   float32
	f64   float64
	text  string
	ref   uintptr
}

type arg struct {
	kind       kind
	class      string
	assignable func(string) bool
}

type candidate struct {
	name   string
	static bool
	params []string
	ret    string
}

func classify(v any) (value, error) {
	switch n := v.(type) {
	case nil:
		return value{kind: kindNil}, nil
	case string:
		return value{kind: kindString, text: n}, nil
	case bool:
		return value{kind: kindBool, b: n}, nil
	case int:
		return classifyInt(int64(n)), nil
	case int8:
		return classifyInt(int64(n)), nil
	case int16:
		return classifyInt(int64(n)), nil
	case int32:
		return classifyInt(int64(n)), nil
	case int64:
		return classifyInt(n), nil
	case uint8:
		return classifyInt(int64(n)), nil
	case float32:
		return value{kind: kindFloat, f32: n}, nil
	case float64:
		return value{kind: kindDouble, f64: n}, nil
	case *Ref:
		if n == nil || n.ptr == 0 {
			return value{kind: kindNil}, nil
		}
		return value{kind: kindRef, class: n.class, ref: n.ptr}, nil
	default:
		return value{}, fmt.Errorf("%w: %T", errArgument, v)
	}
}

func classifyInt(n int64) value {
	if n >= math.MinInt32 && n <= math.MaxInt32 {
		return value{kind: kindInt, i32: int32(n), i64: n}
	}
	return value{kind: kindLong, i64: n}
}

// keepMember reports whether a reflected method is a candidate.
// Bridge methods repeat an erased signature, so they are skipped.
func keepMember(mods int32, wantStatic bool) bool {
	if mods&modBridge != 0 {
		return false
	}
	return (mods&modStatic != 0) == wantStatic
}

func isPrimitive(param string) bool {
	switch param {
	case "boolean", "byte", "char", "short", "int", "long", "float", "double":
		return true
	default:
		return false
	}
}

func scoreArg(param string, a arg) (int, bool) {
	switch a.kind {
	case kindNil:
		if isPrimitive(param) {
			return 0, false
		}
		return 1, true
	case kindBool:
		switch param {
		case "boolean":
			return 4, true
		case "java.lang.Boolean":
			return 3, true
		default:
			return 0, false
		}
	case kindInt:
		switch param {
		case "int":
			return 4, true
		case "java.lang.Integer":
			return 3, true
		case "long":
			return 2, true
		case "java.lang.Long":
			return 1, true
		default:
			return 0, false
		}
	case kindLong:
		switch param {
		case "long":
			return 4, true
		case "java.lang.Long":
			return 3, true
		default:
			return 0, false
		}
	case kindFloat:
		switch param {
		case "float":
			return 4, true
		case "java.lang.Float":
			return 3, true
		case "double":
			return 2, true
		case "java.lang.Double":
			return 1, true
		default:
			return 0, false
		}
	case kindDouble:
		switch param {
		case "double":
			return 4, true
		case "java.lang.Double":
			return 3, true
		default:
			return 0, false
		}
	case kindString:
		switch param {
		case "java.lang.String":
			return 4, true
		case "java.lang.CharSequence":
			return 2, true
		case "java.lang.Object":
			return 1, true
		default:
			return 0, false
		}
	case kindRef:
		if isPrimitive(param) {
			return 0, false
		}
		if param == a.class {
			return 4, true
		}
		if param != "java.lang.Object" && a.assignable != nil && a.assignable(param) {
			return 3, true
		}
		if param == "java.lang.Object" {
			return 1, true
		}
		return 0, false
	default:
		return 0, false
	}
}

type selector struct {
	name   string
	static bool
	args   []arg
}

func pick(cands []candidate, sel selector) (int, error) {
	best := -1
	var matched []int
	for i, c := range cands {
		if c.name != sel.name || c.static != sel.static || len(c.params) != len(sel.args) {
			continue
		}
		score, ok := scoreAll(c, sel.args)
		if !ok {
			continue
		}
		if best < 0 || score > best {
			best = score
			matched = []int{i}
			continue
		}
		if score == best {
			matched = append(matched, i)
		}
	}
	if len(matched) == 0 {
		return -1, fmt.Errorf("%w: %s", errNoMethod, sel.name)
	}
	if len(matched) > 1 {
		sigs := make([]string, len(matched))
		for i, idx := range matched {
			sigs[i] = formatSig(cands[idx])
		}
		return -1, fmt.Errorf("%w: %s", errAmbiguous, strings.Join(sigs, "; "))
	}
	return matched[0], nil
}

func scoreAll(c candidate, args []arg) (int, bool) {
	score := 0
	for i, param := range c.params {
		got, ok := scoreArg(param, args[i])
		if !ok {
			return 0, false
		}
		score += got
	}
	return score, true
}

func formatSig(c candidate) string {
	return c.name + "(" + strings.Join(c.params, ",") + ")"
}

func dotted(name string) string {
	return strings.ReplaceAll(name, "/", ".")
}

func slashed(name string) string {
	return strings.ReplaceAll(name, ".", "/")
}
