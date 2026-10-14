package ndeval

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ndarray"
)

var (
	ErrChoiceOpenNoEvaluator = errors.New("no evaluator")
	ErrResolveEvaluator      = errors.New("evaluator")
)

// Arg is a command flag whose text is an expression of evaluator references.
// Prepare resolves that expression to one ndarray evaluator. x/cmd stores
// Value on the context after Prepare.
//
//	cpu
//	vulkan
//	vulkan:integrated
//	ndeval_opengl
//	vulkan:integrated | cpu
//
// A bare name is an ndeval driver: cpu, vulkan, opengl, metal, or d3d12.
// A full driver id such as ndeval_vulkan is the same reference.
// vulkan:TYPE is the driver id ndeval_vulkan:TYPE.
// The Vulkan evaluator registers that id.
// An integrated choice uses a non-software device when the machine has no integrated GPU.
// Choices separated by | are tried from the left. The first one that opens
// is the evaluator.
type Arg struct {
	expr string
	used string
	ev   ndarray.Evaluator
}

// ArgDefault is the onboard Vulkan evaluator, then the CPU tape.
func (Arg) ArgDefault() string { return "vulkan:integrated | cpu" }

func (a *Arg) Parse(s string) error {
	if _, err := parseExpr(s); err != nil {
		return err
	}
	a.expr = s
	a.used = ""
	a.ev = nil
	return nil
}

// Prepare opens the one evaluator named by the expression.
func (a *Arg) Prepare(ctx context.Context) error {
	if a.ev != nil {
		return nil
	}
	ev, used, err := resolve(ctx, a.expr)
	if err != nil {
		return err
	}
	a.ev = ev
	a.used = used
	return nil
}

// Value is the evaluator opened by Prepare. It is nil before that.
func (a Arg) Value() ndarray.Evaluator { return a.ev }

// Expression is the text that selected the evaluator.
func (a Arg) Expression() string { return a.expr }

// Device is the opened evaluator. A Vulkan evaluator names its physical device.
func (a Arg) Device() string {
	if named, ok := a.ev.(interface{ Name() string }); ok {
		if name := named.Name(); name != "" {
			return name
		}
	}
	if a.used != "" {
		return a.used
	}
	return a.expr
}

// Eval resolves expression to one evaluator.
func Eval(ctx context.Context, expression string) (ndarray.Evaluator, error) {
	ev, _, err := resolve(ctx, expression)
	return ev, err
}

func resolve(ctx context.Context, expression string) (ndarray.Evaluator, string, error) {
	choices, err := parseExpr(expression)
	if err != nil {
		return nil, "", err
	}
	if len(choices) == 0 {
		ev, err := driver.Get[ndarray.Evaluator](ctx)
		return ev, deviceName(ev), err
	}
	var fails []string
	for _, choice := range choices {
		ev, err := choice.open(ctx)
		if err == nil {
			if name := deviceName(ev); name != "" {
				return ev, name, nil
			}
			return ev, choice.label(), nil
		}
		fails = append(fails, err.Error())
	}
	return nil, "", fmt.Errorf("%w %q: %s", ErrResolveEvaluator, expression, strings.Join(fails, "; "))
}

func (c choice) label() string {
	if c.kind == "" {
		return c.name
	}
	return c.name + ":" + c.kind
}

func deviceName(ev ndarray.Evaluator) string {
	if named, ok := ev.(interface{ Name() string }); ok {
		if name := named.Name(); name != "" {
			return name
		}
	}
	return ""
}

type choice struct {
	name string
	kind string
}

func parseExpr(s string) ([]choice, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, "|")
	out := make([]choice, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("%w: empty evaluator choice", cmd.ErrInvalidArgument)
		}
		name, kind, ok := strings.Cut(part, ":")
		name = strings.TrimSpace(name)
		kind = strings.TrimSpace(kind)
		if name == "" || ok && kind == "" || strings.Contains(kind, ":") {
			return nil, fmt.Errorf("%w: %q", cmd.ErrInvalidArgument, part)
		}
		item := choice{name: name, kind: kind}
		if kind != "" && name != "vulkan" && name != "ndeval_vulkan" {
			return nil, fmt.Errorf("%w: %s has no device type", cmd.ErrInvalidArgument, name)
		}
		if kind != "" && !evaluatorRegistered(item.id()) {
			return nil, fmt.Errorf("%w: %s", cmd.ErrInvalidArgument, kind)
		}
		out = append(out, item)
	}
	return out, nil
}

func (c choice) id() string {
	id := c.name
	if !strings.HasPrefix(id, "ndeval_") {
		id = "ndeval_" + id
	}
	if c.kind != "" {
		id += ":" + c.kind
	}
	return id
}

func evaluatorRegistered(id string) bool {
	typ := reflect.TypeFor[ndarray.Evaluator]()
	// RegisteredWeightShape names an interface as package-path.TypeName.
	key := typ.PkgPath() + "." + typ.Name()
	for _, got := range driver.RegisteredWeightShape()[key] {
		if got == id {
			return true
		}
	}
	return false
}

func (c choice) open(ctx context.Context) (ndarray.Evaluator, error) {
	id := c.id()
	handles, err := driver.List[ndarray.Evaluator](ctx)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		if handle.ID == id {
			return handle.Open(ctx)
		}
	}
	return nil, fmt.Errorf("%w %s", ErrChoiceOpenNoEvaluator, id)
}
