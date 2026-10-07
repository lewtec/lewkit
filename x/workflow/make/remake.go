package make

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/workflow"
)

// missedInclude is a makefile named by include that was not on disk.
// GNU make finishes the read, tries to build it, and starts over if it appears.
type missedInclude struct {
	name     string
	abs      string
	from     string
	line     int
	optional bool
}

func (f *File) missingError() error {
	for _, m := range f.missing {
		if m.optional {
			continue
		}
		if _, err := os.Stat(m.abs); err == nil {
			continue
		}
		return fmt.Errorf("make: %s:%d: %s: no such file or directory", m.from, m.line, m.name)
	}
	return fmt.Errorf("make: included makefile was not remade")
}

// remakeMissing builds included makefiles that were absent.
// created is true when a recipe wrote one, so Load reads from scratch again.
func (f *File) remakeMissing(ctx context.Context) (bool, error) {
	created := false
	visiting := map[string]bool{}
	done := map[string]bool{}
	for _, m := range f.missing {
		if _, err := os.Stat(m.abs); err == nil {
			created = true
			continue
		}
		if f.ruleFor(m.name) == nil {
			if m.optional {
				continue
			}
			return false, fmt.Errorf("make: %s:%d: %s: no such file or directory\nmake: %w: %s", m.from, m.line, m.name, ErrNoRule, m.name)
		}
		if err := f.update(ctx, m.name, visiting, done); err != nil {
			if m.optional {
				return false, err
			}
			return false, fmt.Errorf("make: %s:%d: %s: no such file or directory\n%w", m.from, m.line, m.name, err)
		}
		if _, err := os.Stat(m.abs); err == nil {
			created = true
		}
	}
	return created, nil
}

func (f *File) update(ctx context.Context, target string, visiting, done map[string]bool) error {
	if done[target] {
		return nil
	}
	if visiting[target] {
		return fmt.Errorf("make: %w: %s", ErrCycle, target)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	spec := f.ruleFor(target)
	if spec == nil {
		if exists(f.dir, target) {
			done[target] = true
			return nil
		}
		return fmt.Errorf("make: %w: %s", ErrNoRule, target)
	}
	visiting[target] = true
	for _, pre := range spec.prereqs {
		if err := f.update(ctx, pre, visiting, done); err != nil {
			return err
		}
	}
	for _, pre := range spec.order {
		if err := f.update(ctx, pre, visiting, done); err != nil {
			return err
		}
	}
	delete(visiting, target)
	if len(spec.recipe) > 0 && f.stale(target, spec.prereqs) {
		if err := f.runRecipe(ctx, target, spec); err != nil {
			return err
		}
	}
	done[target] = true
	return nil
}

func (f *File) stale(target string, prereqs []string) bool {
	if f.phonies[target] || !exists(f.dir, target) {
		return true
	}
	info, err := os.Stat(workflow.Join(f.dir, target))
	if err != nil {
		return true
	}
	for _, pre := range prereqs {
		if f.phonies[pre] {
			return true
		}
		preInfo, preErr := os.Stat(workflow.Join(f.dir, pre))
		if preErr != nil || !preInfo.ModTime().Before(info.ModTime()) {
			return true
		}
	}
	return false
}

func (f *File) runRecipe(ctx context.Context, target string, spec *resolved) error {
	auto := automatic(f.dir, target, spec.stem, spec.prereqs)
	var cmds []workflow.Command
	var env []string
	err := f.withTargetVars(target, func() error {
		var err error
		cmds, err = f.recipes(spec.recipe, auto)
		if err != nil {
			return err
		}
		env, err = f.recipeEnv()
		return err
	})
	if err != nil {
		return err
	}
	for _, cmd := range cmds {
		cmd.Dir = f.dir
		cmd.Env = env
		if err := cmd.Run(ctx, nil); err != nil {
			return fmt.Errorf("%s: %w", target, err)
		}
	}
	return nil
}
