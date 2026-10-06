// Package ninja reads a ninja build file into a workflow graph.
//
// Rules, build edges, variables, includes, and gcc depfiles are
// enough for a small Linux build. Commands run through workflow,
// so a conda compiler on PATH is the one the edge invokes.
package ninja

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/workflow"
)

var (
	// ErrSyntax means the ninja file could not be parsed.
	ErrSyntax = errors.New("syntax")
	// ErrCycle means variables expand through themselves.
	ErrCycle = errors.New("cycle")
)

type scope struct {
	parent *scope
	vars   map[string]string
}

func (s *scope) get(name string) (string, bool) {
	for s != nil {
		if v, ok := s.vars[name]; ok {
			return v, true
		}
		s = s.parent
	}
	return "", false
}

type rule struct {
	vars map[string]string
}

type edge struct {
	outs     []string
	explicit []string
	implicit []string
	order    []string
	rule     string
	vars     map[string]string
	file     string
	line     int
}

// File is a parsed ninja file.
type File struct {
	dir      string
	scope    *scope
	rules    map[string]*rule
	edges    []*edge
	defaults []string
}

// Load reads path. work is the directory commands run in; an empty work
// uses the file's directory.
func Load(ctx context.Context, path, work string) (*File, error) {
	if ctx == nil {
		return nil, errors.New("ninja: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if work == "" {
		work = filepath.Dir(abs)
	} else {
		work, err = filepath.Abs(work)
		if err != nil {
			return nil, err
		}
	}
	f := &File{
		dir:   work,
		scope: &scope{vars: map[string]string{}},
		rules: map[string]*rule{},
	}
	if err := f.parseFile(ctx, abs, f.scope); err != nil {
		return nil, err
	}
	return f, nil
}

// Graph resolves the file into a workflow graph. An empty target list
// uses default statements, or the first build edge.
func (f *File) Graph(targets []string) (workflow.Graph, error) {
	byOutput := map[string]string{}
	for _, edge := range f.edges {
		if len(edge.outs) == 0 {
			return workflow.Graph{}, fmt.Errorf("ninja: %s:%d: %w: build has no output", edge.file, edge.line, ErrSyntax)
		}
		for _, out := range edge.outs {
			if prev, ok := byOutput[out]; ok {
				return workflow.Graph{}, fmt.Errorf("ninja: %w: duplicate output %s of %s", ErrSyntax, out, prev)
			}
			byOutput[out] = edge.outs[0]
		}
	}
	alias := map[string]string{}
	var steps []workflow.Step
	for _, edge := range f.edges {
		step, err := f.step(edge, byOutput)
		if err != nil {
			return workflow.Graph{}, err
		}
		steps = append(steps, step)
		for _, out := range edge.outs[1:] {
			alias[out] = edge.outs[0]
		}
	}
	defaults := targets
	if len(defaults) == 0 {
		defaults = append([]string{}, f.defaults...)
	}
	if len(defaults) == 0 && len(f.edges) > 0 {
		defaults = []string{f.edges[0].outs[0]}
	}
	if len(defaults) == 0 {
		return workflow.Graph{}, fmt.Errorf("ninja: %w", workflow.ErrNoTarget)
	}
	return workflow.Graph{Dir: f.dir, Steps: steps, Defaults: defaults, Alias: alias}, nil
}

func (f *File) step(edge *edge, byOutput map[string]string) (workflow.Step, error) {
	resolve := func(name string) (string, bool, bool) {
		switch name {
		case "in":
			return strings.Join(edge.explicit, " "), true, true
		case "out":
			return strings.Join(edge.outs, " "), true, true
		case "in_newline":
			return strings.Join(edge.explicit, "\n"), true, true
		}
		if v, ok := edge.vars[name]; ok {
			return v, false, true
		}
		if rule := f.rules[edge.rule]; rule != nil {
			if v, ok := rule.vars[name]; ok {
				return v, false, true
			}
		}
		if v, ok := f.scope.get(name); ok {
			return v, false, true
		}
		return "", false, false
	}
	command := ""
	if rule := f.rules[edge.rule]; rule != nil {
		if v, ok := edge.vars["command"]; ok {
			command = v
		} else if v, ok := rule.vars["command"]; ok {
			command = v
		}
	}
	expanded, err := expand(command, resolve, false)
	if err != nil {
		return workflow.Step{}, fmt.Errorf("ninja: %s:%d: %w", edge.file, edge.line, err)
	}
	descRaw := ""
	if rule := f.rules[edge.rule]; rule != nil {
		if v, ok := edge.vars["description"]; ok {
			descRaw = v
		} else if v, ok := rule.vars["description"]; ok {
			descRaw = v
		}
	}
	desc, err := expand(descRaw, resolve, false)
	if err != nil {
		return workflow.Step{}, err
	}
	desc = strings.TrimSpace(desc)
	phony := edge.rule == "phony" || strings.TrimSpace(expanded) == ""
	var tasks []workflow.Task
	if !phony {
		tasks = []workflow.Task{workflow.Command{Text: expanded, Dir: f.dir}}
		if desc == "" {
			desc = expanded
		}
	}
	if desc == "" {
		desc = edge.outs[0]
	}
	inputs := append(append([]string{}, edge.explicit...), edge.implicit...)
	if deps, err := f.depfile(edge, resolve); err != nil {
		return workflow.Step{}, err
	} else {
		inputs = append(inputs, deps...)
	}
	seen := map[string]bool{}
	var deps []string
	addDep := func(path string) {
		name, ok := byOutput[path]
		if !ok || name == edge.outs[0] || seen[name] {
			return
		}
		seen[name] = true
		deps = append(deps, name)
	}
	for _, in := range inputs {
		addDep(in)
	}
	for _, in := range edge.order {
		addDep(in)
	}
	step := workflow.Step{
		Name:   edge.outs[0],
		Deps:   deps,
		Inputs: inputs,
		Dir:    f.dir,
		Tasks:  tasks,
		Desc:   desc,
		Phony:  phony,
	}
	if !phony {
		step.Outputs = append([]string{}, edge.outs...)
	}
	return step, nil
}

func (f *File) depfile(edge *edge, resolve func(string) (string, bool, bool)) ([]string, error) {
	raw := ""
	if v, ok := edge.vars["depfile"]; ok {
		raw = v
	} else if rule := f.rules[edge.rule]; rule != nil {
		raw = rule.vars["depfile"]
	}
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	name, err := expand(raw, resolve, false)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	data, err := os.ReadFile(workflow.Join(f.dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseDepfile(string(data)), nil
}

func parseDepfile(src string) []string {
	src = strings.ReplaceAll(src, "\\\n", " ")
	src = strings.ReplaceAll(src, "\\\r\n", " ")
	var deps []string
	for _, line := range strings.Split(src, "\n") {
		_, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		for _, word := range strings.Fields(rest) {
			if word != "\\" {
				deps = append(deps, word)
			}
		}
	}
	return deps
}
