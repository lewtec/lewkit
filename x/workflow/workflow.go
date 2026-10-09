// Package workflow runs a task graph on a taskgroup session.
//
// A step lists tasks. A shell command, a download, an extract, and an
// in-process function are the kinds. Make and ninja both produce a
// graph of commands.
// Run schedules each step with taskgroup.Go and returns a Future for
// those steps. The progress view shows the work. A step with outputs
// is skipped when those files are newer than its inputs and no
// dependency ran. Commands inherit the process environment, which is
// how a conda compiler on PATH or in CC is the one that compiles.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
)

var (
	// ErrNoTarget means Run was given no target and the graph has no default.
	ErrNoTarget = errors.New("no target")
	// ErrCycle means steps depend on themselves.
	ErrCycle = errors.New("cycle")
	// ErrUnknownStep means a target or dependency names no step.
	ErrUnknownStep = errors.New("unknown step")
	// ErrDuplicate means two steps share a name.
	ErrDuplicate = errors.New("duplicate step")
)

// Step is one node. Deps are step names. Inputs and Outputs are paths,
// relative to Dir or Graph.Dir unless absolute. Tasks run in order when
// the step is out of date. Phony steps run every time. An empty Pool
// with a task uses the CPU pool. A step with no tasks is Control.
type Step struct {
	Name    string
	Deps    []string
	Inputs  []string
	Outputs []string
	Dir     string
	Tasks   []Task
	Desc    string
	Phony   bool
	Pool    taskgroup.PoolKind
}

// Graph is the build the frontends hand to Run.
// Alias maps a secondary name, such as an extra ninja output, onto a step.
type Graph struct {
	Dir      string
	Steps    []Step
	Defaults []string
	Alias    map[string]string
}

// Future is the steps Run scheduled.
// Wait blocks until those steps finish. It does not wait for the rest
// of the session, so the task that scheduled the graph can wait for it.
type Future struct {
	session *taskgroup.Session
	ids     []taskgroup.ID
}

// Wait blocks until every scheduled step has finished and returns the
// first step error. A future with no steps returns nil. A cancelled
// ctx does not cut the wait short; the step context stops the step.
func (f Future) Wait(ctx context.Context) error {
	if len(f.ids) == 0 {
		return nil
	}
	if ctx == nil {
		return errors.New("workflow: nil context")
	}
	ctx = context.WithoutCancel(ctx)
	var err error
	for _, id := range f.ids {
		werr := f.session.WaitTask(ctx, id)
		if werr != nil && err == nil {
			err = werr
		}
	}
	return err
}

// Run builds targets. An empty list uses Graph.Defaults.
// The future is those steps. A session already on ctx is used and not
// waited on; Wait on the future waits for the steps. Without a session,
// Run starts one and waits, and that wait error is Run's error.
func Run(ctx context.Context, g Graph, targets []string) (Future, error) {
	if ctx == nil {
		return Future{}, errors.New("workflow: nil context")
	}
	if err := ctx.Err(); err != nil {
		return Future{}, err
	}
	order, byName, err := g.closure(targets)
	if err != nil {
		return Future{}, err
	}
	var fut Future
	err = taskgroup.WithSession(ctx, func(ctx context.Context) error {
		var schedErr error
		fut, schedErr = schedule(ctx, g.Dir, order, byName, g.Alias)
		return schedErr
	})
	return fut, err
}

func schedule(ctx context.Context, dir string, order []Step, byName map[string]Step, alias map[string]string) (Future, error) {
	rt := &runtime{rebuilt: map[string]bool{}}
	ids := map[string]taskgroup.ID{}
	scheduled := make([]taskgroup.ID, 0, len(order))
	for _, step := range order {
		deps := make([]taskgroup.ID, 0, len(step.Deps))
		for _, d := range step.Deps {
			name, err := resolveAlias(d, byName, alias)
			if err != nil {
				return Future{}, err
			}
			id, ok := ids[name]
			if !ok {
				return Future{}, fmt.Errorf("workflow: %w: %s", ErrUnknownStep, d)
			}
			deps = append(deps, id)
		}
		step := step
		id := taskgroup.Go(ctx, step.Name, step.pool(), func(ctx context.Context, st *taskgroup.Status) error {
			return runStep(ctx, dir, step, st, rt)
		}, deps...)
		ids[step.Name] = id
		scheduled = append(scheduled, id)
	}
	return Future{session: taskgroup.MustFromContext(ctx), ids: scheduled}, nil
}

func (g Graph) closure(targets []string) ([]Step, map[string]Step, error) {
	byName := make(map[string]Step, len(g.Steps))
	for _, step := range g.Steps {
		if step.Name == "" {
			return nil, nil, errors.New("workflow: empty step name")
		}
		if _, ok := byName[step.Name]; ok {
			return nil, nil, fmt.Errorf("workflow: %w: %s", ErrDuplicate, step.Name)
		}
		byName[step.Name] = step
	}
	if len(targets) == 0 {
		targets = g.Defaults
	}
	if len(targets) == 0 {
		return nil, nil, fmt.Errorf("workflow: %w", ErrNoTarget)
	}
	state := map[string]int{}
	var order []Step
	var walk func(string) error
	walk = func(name string) error {
		canon, err := resolveAlias(name, byName, g.Alias)
		if err != nil {
			return err
		}
		switch state[canon] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("workflow: %w: %s", ErrCycle, canon)
		}
		step := byName[canon]
		state[canon] = 1
		for _, dep := range step.Deps {
			if err := walk(dep); err != nil {
				return err
			}
		}
		state[canon] = 2
		order = append(order, step)
		return nil
	}
	for _, target := range targets {
		if err := walk(target); err != nil {
			return nil, nil, err
		}
	}
	return order, byName, nil
}

func resolveAlias(name string, byName map[string]Step, alias map[string]string) (string, error) {
	seen := map[string]bool{}
	for {
		if seen[name] {
			return "", fmt.Errorf("workflow: %w: %s", ErrCycle, name)
		}
		seen[name] = true
		if _, ok := byName[name]; ok {
			return name, nil
		}
		next, ok := alias[name]
		if !ok {
			return "", fmt.Errorf("workflow: %w: %s", ErrUnknownStep, name)
		}
		name = next
	}
}

func (s Step) pool() taskgroup.PoolKind {
	if s.Pool != taskgroup.Control {
		return s.Pool
	}
	if len(s.Tasks) > 0 {
		return taskgroup.CPU
	}
	return taskgroup.Control
}

func (s Step) title() string {
	if s.Desc != "" {
		return s.Desc
	}
	for _, task := range s.Tasks {
		labeled, ok := task.(fmt.Stringer)
		if !ok {
			continue
		}
		if text := strings.TrimSpace(labeled.String()); text != "" {
			return text
		}
	}
	return s.Name
}

// Join joins dir with name. dir is an OS path, such as a CLI directory
// or a makefile location. name is a slash-separated file name. An
// absolute name is returned unchanged: a host path is not stored in a Path.
func Join(dir, name string) string {
	if name == "" {
		return ""
	}
	p := lewpath.New(name)
	if p.IsAbs() {
		return name
	}
	if dir == "" {
		return name
	}
	if p.String() == "." {
		return dir
	}
	return filepath.Join(dir, filepath.FromSlash(p.String()))
}

func mergeEnv(base, overlay []string) []string {
	if len(overlay) == 0 {
		return base
	}
	index := map[string]int{}
	out := append([]string(nil), base...)
	for _, kv := range overlay {
		key, _, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			continue
		}
		if i, found := index[key]; found {
			out[i] = kv
			continue
		}
		if i := envIndex(out, key); i >= 0 {
			index[key] = i
			out[i] = kv
			continue
		}
		index[key] = len(out)
		out = append(out, kv)
	}
	return out
}

func envIndex(env []string, key string) int {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return i
		}
	}
	return -1
}

func mkdirOutputs(dir string, outs []string) error {
	for _, out := range outs {
		parent := outputParent(dir, out)
		if parent == "" || parent == "." {
			continue
		}
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// outputParent is the host directory that must exist before writing name.
// A relative name's parent is a Path. Absolute names, and names io/fs
// rejects, use the host path.
func outputParent(dir, name string) string {
	if name == "" {
		return ""
	}
	p := lewpath.New(name)
	if p.IsAbs() || !p.Valid() {
		return filepath.Dir(Join(dir, name))
	}
	parent := p.Parent()
	if parent.String() == "." {
		if dir == "" {
			return "."
		}
		return dir
	}
	if dir == "" {
		return parent.String()
	}
	return filepath.Join(dir, filepath.FromSlash(parent.String()))
}
