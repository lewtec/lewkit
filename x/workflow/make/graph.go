package make

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/workflow"
)

type resolved struct {
	prereqs []string
	order   []string
	recipe  []string
	stem    string
}

type builder struct {
	f        *File
	steps    map[string]*workflow.Step
	order    []string
	visiting map[string]bool
	done     map[string]bool
}

// Graph resolves targets into a workflow graph. An empty list builds the
// default goal.
func (f *File) Graph(ctx context.Context, targets []string) (workflow.Graph, error) {
	if ctx == nil {
		return workflow.Graph{}, errors.New("make: nil context")
	}
	f.ctx = ctx
	if len(targets) == 0 {
		var err error
		targets, err = f.defaultGoals()
		if err != nil {
			return workflow.Graph{}, err
		}
	}
	if len(targets) == 0 {
		return workflow.Graph{}, fmt.Errorf("make: %w", workflow.ErrNoTarget)
	}
	b := &builder{
		f:        f,
		steps:    map[string]*workflow.Step{},
		visiting: map[string]bool{},
		done:     map[string]bool{},
	}
	for _, target := range targets {
		if err := b.need(target); err != nil {
			return workflow.Graph{}, err
		}
	}
	g := workflow.Graph{Dir: f.dir, Defaults: append([]string{}, targets...)}
	for _, name := range b.order {
		g.Steps = append(g.Steps, *b.steps[name])
	}
	return g, nil
}

func (f *File) defaultGoals() ([]string, error) {
	if v, ok := f.vars[".DEFAULT_GOAL"]; ok && strings.TrimSpace(v.value) != "" {
		text := v.value
		var err error
		if !v.simple {
			text, err = f.expand(v.value, nil)
			if err != nil {
				return nil, err
			}
		}
		return strings.Fields(text), nil
	}
	if f.first == "" {
		return nil, nil
	}
	return []string{f.first}, nil
}

func (b *builder) need(target string) error {
	if b.done[target] {
		return nil
	}
	if b.visiting[target] {
		return fmt.Errorf("make: %w: %s", ErrCycle, target)
	}
	b.visiting[target] = true
	spec := b.f.ruleFor(target)
	if spec == nil {
		if exists(b.f.dir, target) {
			b.visiting[target] = false
			b.done[target] = true
			return nil
		}
		return fmt.Errorf("make: %w: %s", ErrNoRule, target)
	}
	for _, pre := range spec.prereqs {
		if err := b.need(pre); err != nil {
			return err
		}
	}
	for _, pre := range spec.order {
		if err := b.need(pre); err != nil {
			return err
		}
	}
	b.visiting[target] = false
	b.done[target] = true
	return b.emit(target, spec)
}

func (f *File) ruleFor(target string) *resolved {
	var extraPre, extraOrd, recipe []string
	stem := ""
	if rule, ok := f.explicit[target]; ok {
		extraPre = rule.prereqs
		extraOrd = rule.order
		recipe = rule.recipe
		stem = rule.stem
	}
	if len(recipe) > 0 {
		return &resolved{prereqs: uniq(extraPre), order: uniq(extraOrd), recipe: recipe, stem: stem}
	}
	if pat := f.matchPattern(target); pat != nil {
		// The pattern's own prerequisites stay first, so $< is the source file.
		// Extra prerequisites named on the target come after them.
		pat.prereqs = uniq(append(append([]string{}, pat.prereqs...), extraPre...))
		pat.order = uniq(append(append([]string{}, pat.order...), extraOrd...))
		return pat
	}
	if _, ok := f.explicit[target]; ok {
		return &resolved{prereqs: uniq(extraPre), order: uniq(extraOrd), stem: stem}
	}
	return nil
}

func (f *File) matchPattern(target string) *resolved {
	stack := map[string]bool{target: true}
	for _, pat := range f.patterns {
		if pat.body == nil {
			continue
		}
		stem, ok := stemOf(pat.target, target)
		if !ok {
			continue
		}
		prereqs := applyStem(pat.body.prereqs, stem)
		good := f.prereqsMakeable(prereqs, stack, pat.target == "%")
		if !good {
			continue
		}
		return &resolved{
			prereqs: prereqs,
			order:   applyStem(pat.body.order, stem),
			recipe:  pat.body.recipe,
			stem:    stem,
		}
	}
	return nil
}

func (f *File) canMake(target string, visiting map[string]bool) bool {
	if len(visiting) > 32 || visiting[target] {
		return false
	}
	if _, ok := f.explicit[target]; ok {
		return true
	}
	if exists(f.dir, target) {
		return true
	}
	visiting[target] = true
	defer delete(visiting, target)
	for _, pat := range f.patterns {
		if pat.body == nil {
			continue
		}
		stem, ok := stemOf(pat.target, target)
		if !ok {
			continue
		}
		prereqs := applyStem(pat.body.prereqs, stem)
		if f.prereqsMakeable(prereqs, visiting, pat.target == "%") {
			return true
		}
	}
	return false
}

// prereqsMakeable reports whether a pattern rule can build its prerequisites.
// A match-anything pattern only accepts prerequisites that already exist or
// have an explicit rule, so `%: %.o` cannot chain through itself.
func (f *File) prereqsMakeable(prereqs []string, visiting map[string]bool, terminal bool) bool {
	if len(prereqs) == 0 {
		return false
	}
	for _, pre := range prereqs {
		if terminal {
			if _, ok := f.explicit[pre]; ok || exists(f.dir, pre) {
				continue
			}
			return false
		}
		if !f.canMake(pre, visiting) {
			return false
		}
	}
	return true
}

func (b *builder) emit(target string, spec *resolved) error {
	auto := automatic(b.f.dir, target, spec.stem, spec.prereqs)
	var cmds []workflow.Command
	var env []string
	err := b.f.withTargetVars(target, func() error {
		var err error
		cmds, err = b.f.recipes(spec.recipe, auto)
		if err != nil {
			return err
		}
		env, err = b.f.recipeEnv()
		return err
	})
	if err != nil {
		return err
	}
	var deps []string
	for _, pre := range spec.prereqs {
		if _, ok := b.steps[pre]; ok {
			deps = append(deps, pre)
		}
	}
	for _, pre := range spec.order {
		if _, ok := b.steps[pre]; ok {
			deps = append(deps, pre)
		}
	}
	phony := b.f.phonies[target]
	if len(cmds) == 0 && !exists(b.f.dir, target) {
		phony = true
	}
	inputs := append([]string{}, spec.prereqs...)
	if !phony {
		inputs = append(inputs, b.f.rel(b.f.path))
	}
	tasks := make([]workflow.Task, len(cmds))
	for i, cmd := range cmds {
		cmd.Dir = b.f.dir
		cmd.Env = env
		tasks[i] = cmd
	}
	step := &workflow.Step{
		Name:   target,
		Deps:   uniq(deps),
		Inputs: uniq(inputs),
		Dir:    b.f.dir,
		Tasks:  tasks,
		Phony:  phony,
		Desc:   describe(cmds, target),
	}
	if !phony {
		step.Outputs = []string{target}
	}
	b.steps[target] = step
	b.order = append(b.order, target)
	return nil
}

func (f *File) recipes(lines []string, auto map[string]string) ([]workflow.Command, error) {
	if f.oneshell {
		var b strings.Builder
		ignore := false
		for i, raw := range lines {
			text, ign := recipeText(raw)
			if i == 0 {
				ignore = ign
			}
			expanded, err := f.expand(text, auto)
			if err != nil {
				return nil, err
			}
			var quiet bool
			expanded, quiet = recipeText(strings.TrimLeft(expanded, " \t"))
			if quiet {
				ignore = true
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(expanded)
		}
		if b.Len() == 0 {
			return nil, nil
		}
		return []workflow.Command{{Text: b.String(), Ignore: ignore}}, nil
	}
	var out []workflow.Command
	for _, raw := range lines {
		text, ignore := recipeText(raw)
		expanded, err := f.expand(text, auto)
		if err != nil {
			return nil, err
		}
		// $(Q) expands to @. GNU make strips that prefix after expansion.
		expanded, ign := recipeText(strings.TrimLeft(expanded, " \t"))
		if ign {
			ignore = true
		}
		if strings.TrimSpace(expanded) == "" {
			continue
		}
		out = append(out, workflow.Command{Text: expanded, Ignore: ignore})
	}
	return out, nil
}

func recipeText(line string) (string, bool) {
	i := 0
	ignore := false
	for i < len(line) {
		switch line[i] {
		case '@', '+':
			i++
		case '-':
			ignore = true
			i++
		default:
			return line[i:], ignore
		}
	}
	return "", ignore
}

func describe(cmds []workflow.Command, name string) string {
	for _, cmd := range cmds {
		if text := strings.TrimSpace(cmd.Text); text != "" {
			return text
		}
	}
	return name
}

func automatic(dir, target, stem string, prereqs []string) map[string]string {
	auto := map[string]string{
		"@": target,
		"^": strings.Join(uniq(prereqs), " "),
		"+": strings.Join(prereqs, " "),
		"*": stem,
	}
	if len(prereqs) > 0 {
		auto["<"] = prereqs[0]
	}
	var newer []string
	info, err := os.Stat(workflow.Join(dir, target))
	for _, pre := range prereqs {
		preInfo, preErr := os.Stat(workflow.Join(dir, pre))
		if preErr != nil || err != nil || preInfo.ModTime().After(info.ModTime()) {
			newer = append(newer, pre)
		}
	}
	auto["?"] = strings.Join(newer, " ")
	auto["@D"], auto["@F"] = splitDF(target)
	auto["<D"], auto["<F"] = splitDF(auto["<"])
	auto["*D"], auto["*F"] = splitDF(stem)
	auto["^D"], auto["^F"] = splitDF(auto["^"])
	return auto
}

func splitDF(name string) (string, string) {
	if name == "" {
		return ".", ""
	}
	return dirOf(name), notDir(name)
}

func (f *File) rel(full string) string {
	rel, err := filepath.Rel(f.dir, full)
	if err != nil {
		return full
	}
	return lewpath.New(filepath.ToSlash(rel)).String()
}

func (f *File) recipeEnv() ([]string, error) {
	names := map[string]struct{}{}
	if f.exportAll {
		for name := range f.vars {
			names[name] = struct{}{}
		}
	}
	for name := range f.exported {
		names[name] = struct{}{}
	}
	if len(names) == 0 {
		return nil, nil
	}
	overlay := make([]string, 0, len(names))
	for name := range names {
		if f.unexported[name] {
			continue
		}
		v, ok := f.vars[name]
		if !ok {
			continue
		}
		value := v.value
		if !v.simple {
			var err error
			value, err = f.expand(v.value, nil)
			if err != nil {
				return nil, err
			}
		}
		overlay = append(overlay, name+"="+value)
	}
	sort.Strings(overlay)
	if len(overlay) == 0 {
		return nil, nil
	}
	return mergeEnv(os.Environ(), overlay), nil
}

func mergeEnv(base, overlay []string) []string {
	out := append([]string(nil), base...)
	index := map[string]int{}
	for i, kv := range out {
		key, _, ok := strings.Cut(kv, "=")
		if ok {
			index[key] = i
		}
	}
	for _, kv := range overlay {
		key, _, ok := strings.Cut(kv, "=")
		if !ok || key == "" {
			continue
		}
		if i, found := index[key]; found {
			out[i] = kv
			continue
		}
		index[key] = len(out)
		out = append(out, kv)
	}
	return out
}
