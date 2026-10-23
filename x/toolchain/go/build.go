package gotool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

const (
	// LinkStep is the step [Plan] adds for [Image].
	LinkStep = "toolchain:link"

	actionEnv  = "LEWKIT_TOOLCHAIN_ACTION"
	requestEnv = "LEWKIT_TOOLCHAIN_REQUEST"
)

var (
	// ErrNoWork means Plan was given no units and no image.
	ErrNoWork = errors.New("toolchain: nothing to build")
	// ErrEmptyUnit means a unit has no import path, files, archive, or importcfg.
	ErrEmptyUnit = errs.Wrap(errs.ErrEmpty, "toolchain: empty unit")
	// ErrEmptyImage means the link has no output, archive, importcfg, or mode.
	ErrEmptyImage = errs.Wrap(errs.ErrEmpty, "toolchain: empty link")
	// ErrDuplicate means two units share an import path, or a unit is named [LinkStep].
	ErrDuplicate = errs.Wrap(errs.ErrDuplicate, "toolchain: duplicate unit")
	// ErrUnknownDep means a dependency is not a unit in this build.
	ErrUnknownDep = errs.Wrap(errs.ErrUnknown, "toolchain: unknown dependency")
)

// Unit is one package the graph compiles.
// Deps are other units in the same graph. ImportCfg names every import the
// compiler will see, including those Deps, at the archive paths Plan records.
type Unit struct {
	ImportPath string
	Files      []string
	Deps       []string
	ImportCfg  string
	EmbedCfg   string
	Lang       string
	Archive    string
	Shared     bool
	// Asm is the package's .s and .syso files. [Plan] rejects a unit
	// that sets this. [BuildModule] assembles [Unit.SFiles] instead.
	Asm []string
	// SFiles and SysoFiles are assembled into Archive.
	SFiles    []string
	SysoFiles []string
	// Std sets the compiler and assembler -std flag.
	Std bool
	// NoComplete omits -complete. Assembly and a few standard packages
	// supply bodies from outside the Go files.
	NoComplete bool
	// Include is the assembler's pkg/include directory.
	Include string
	// Reuse means Archive is already built. The step does not compile.
	Reuse bool
	// GOOS and GOARCH are the target of the archive in [Unit.Object].
	// Empty values inherit the [Target] passed to [BuildObjects].
	// A set value is the child process target. buildcfg reads it at start.
	GOOS   string
	GOARCH string
	// Trimpath is the compiler -trimpath prefix. Empty leaves paths as they are.
	Trimpath string
	// SymABIs is a -symabis file. A non-empty value omits -complete so
	// assembly packed into the archive later can supply the bodies.
	// The toolchain does not assemble those files.
	SymABIs string
}

// Image is the link step. Dep is the unit whose archive is Archive.
// An empty Dep waits for every unit. Ldflags are extra linker arguments.
// Bare metal passes [github.com/lewtec/lewkit/x/toolchain/tamago.LinkFlags].
type Image struct {
	Out       string
	ImportCfg string
	Mode      string
	Archive   string
	Dep       string
	Ldflags   []string
	// GOOS and GOARCH are the child process target. Empty keeps the
	// caller's target. buildcfg reads them at process start.
	GOOS   string
	GOARCH string
}

// Plan builds the workflow graph for units and an optional image.
// Each unit is a step. Deps are the units it imports. The image is
// [LinkStep] and waits on Dep, or on every unit when Dep is empty.
func Plan(units []Unit, image *Image) (workflow.Graph, error) {
	if len(units) == 0 && image == nil {
		return workflow.Graph{}, ErrNoWork
	}
	byPath := make(map[string]Unit, len(units))
	for _, u := range units {
		if len(u.Asm) > 0 {
			return workflow.Graph{}, fmt.Errorf("%w: %s", ErrNoAsm, u.ImportPath)
		}
		if u.ImportPath == "" || u.Archive == "" || u.ImportCfg == "" || len(u.Files) == 0 {
			return workflow.Graph{}, fmt.Errorf("%w: %s", ErrEmptyUnit, u.ImportPath)
		}
		if u.ImportPath == LinkStep {
			return workflow.Graph{}, fmt.Errorf("%w: %s", ErrDuplicate, u.ImportPath)
		}
		if _, ok := byPath[u.ImportPath]; ok {
			return workflow.Graph{}, fmt.Errorf("%w: %s", ErrDuplicate, u.ImportPath)
		}
		byPath[u.ImportPath] = u
	}
	steps := make([]workflow.Step, 0, len(units)+1)
	names := make([]string, 0, len(units))
	for _, u := range units {
		for _, dep := range u.Deps {
			if _, ok := byPath[dep]; !ok {
				return workflow.Graph{}, fmt.Errorf("%w: %s", ErrUnknownDep, dep)
			}
		}
		steps = append(steps, unitStep(u, byPath))
		names = append(names, u.ImportPath)
	}
	defaults := names
	if image != nil {
		if image.Out == "" || image.Archive == "" || image.ImportCfg == "" || image.Mode == "" {
			return workflow.Graph{}, ErrEmptyImage
		}
		deps := append([]string(nil), names...)
		if image.Dep != "" {
			if _, ok := byPath[image.Dep]; !ok {
				return workflow.Graph{}, fmt.Errorf("%w: %s", ErrUnknownDep, image.Dep)
			}
			deps = []string{image.Dep}
		}
		steps = append(steps, imageStep(*image, deps))
		defaults = []string{LinkStep}
	}
	return workflow.Graph{Steps: steps, Defaults: defaults}, nil
}

// Build runs [Plan] on x/workflow.
// Each package is one step, and the link is the step that waits on them.
// A step calls [Compile] or [Link], which run in their own process, and
// reports that package on the taskgroup status. A session already on ctx,
// including the progress view entry.Run starts, shows those rows.
func Build(ctx context.Context, units []Unit, image *Image) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	g, err := Plan(units, image)
	if err != nil {
		return err
	}
	fut, err := workflow.Run(ctx, g, nil)
	if err != nil {
		return err
	}
	return fut.Wait(ctx)
}

// TestExit runs [Child] when this process is a toolchain step.
// Otherwise it runs m and exits with that code.
func TestExit(m interface{ Run() int }) {
	if handled, code := stepExit(); handled {
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func stepExit() (bool, int) {
	handled, err := Child()
	if !handled {
		return false, 0
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	return true, 0
}

// Child runs the one compile or link this process was started to do.
// [Compile] and [Link] start that process. It reports whether this
// process is that step.
func Child() (bool, error) {
	action := os.Getenv(actionEnv)
	if action == "" {
		return false, nil
	}
	path := os.Getenv(requestEnv)
	if path == "" {
		return true, ErrToolchainMissingRequest
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return true, err
	}
	var req childRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return true, err
	}
	switch action {
	case "compile":
		return true, compileHere(req.ImportPath, req.Out, req.ImportCfg, req.EmbedCfg, req.Lang, req.Shared, req.Std, req.NoComplete, req.Trimpath, req.SymABIs, req.Asmhdr, req.Files)
	case "asm":
		return true, asmHere(req)
	case "link":
		return true, linkHere(req.Out, req.ImportCfg, req.Mode, req.Archive, req.Ldflags...)
	default:
		return true, fmt.Errorf("%w %s", ErrToolchainUnknownAction, action)
	}
}

func unitStep(u Unit, byPath map[string]Unit) workflow.Step {
	var inputs []string
	if !u.Reuse {
		inputs = append(inputs, u.Files...)
		inputs = append(inputs, u.SFiles...)
		inputs = append(inputs, u.SysoFiles...)
		inputs = append(inputs, u.ImportCfg)
		if u.EmbedCfg != "" {
			inputs = append(inputs, u.EmbedCfg)
		}
		for _, dep := range u.Deps {
			inputs = append(inputs, byPath[dep].Archive)
		}
	}
	unit := u
	return workflow.Step{
		Name:    unit.ImportPath,
		Deps:    append([]string(nil), unit.Deps...),
		Inputs:  inputs,
		Outputs: []string{unit.Archive},
		Desc:    "compile " + unit.ImportPath,
		Tasks:   []workflow.Task{compileTask(unit)},
	}
}

func imageStep(image Image, deps []string) workflow.Step {
	return workflow.Step{
		Name:    LinkStep,
		Deps:    deps,
		Inputs:  []string{image.Archive, image.ImportCfg},
		Outputs: []string{image.Out},
		Desc:    "link " + image.Mode,
		Tasks:   []workflow.Task{linkTask(image)},
	}
}

func compileTask(u Unit) workflow.Task {
	return workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		done := st.Unit()
		defer done()
		st.Update(u.ImportPath)
		if u.Reuse {
			return nil
		}
		if len(u.SFiles) > 0 || len(u.SysoFiles) > 0 {
			return buildAssembled(ctx, u)
		}
		part := partPath(u.Archive)
		err := spawn(ctx, "compile", childRequest{
			ImportPath: u.ImportPath,
			Out:        part,
			ImportCfg:  u.ImportCfg,
			EmbedCfg:   u.EmbedCfg,
			Lang:       u.Lang,
			Shared:     u.Shared,
			Std:        u.Std,
			NoComplete: u.NoComplete,
			Files:      u.Files,
			GOOS:       u.GOOS,
			GOARCH:     u.GOARCH,
			Trimpath:   u.Trimpath,
			SymABIs:    u.SymABIs,
		})
		if err != nil {
			os.Remove(part)
			return err
		}
		if err := os.Rename(part, u.Archive); err != nil {
			os.Remove(part)
			return err
		}
		return nil
	})
}

func partPath(archive string) string {
	return fmt.Sprintf("%s.%d.part", archive, os.Getpid())
}

func linkTask(image Image) workflow.Task {
	return workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		done := st.Unit()
		defer done()
		st.Update(image.Mode)
		return spawn(ctx, "link", childRequest{
			Out:       image.Out,
			ImportCfg: image.ImportCfg,
			Mode:      image.Mode,
			Archive:   image.Archive,
			Ldflags:   image.Ldflags,
			GOOS:      image.GOOS,
			GOARCH:    image.GOARCH,
		})
	})
}

type childRequest struct {
	Action     string   `json:"action"`
	ImportPath string   `json:"importPath,omitempty"`
	Out        string   `json:"out"`
	ImportCfg  string   `json:"importCfg"`
	EmbedCfg   string   `json:"embedCfg,omitempty"`
	Lang       string   `json:"lang,omitempty"`
	Shared     bool     `json:"shared,omitempty"`
	Files      []string `json:"files,omitempty"`
	Mode       string   `json:"mode,omitempty"`
	Archive    string   `json:"archive,omitempty"`
	Ldflags    []string `json:"ldflags,omitempty"`
	GOOS       string   `json:"goos,omitempty"`
	GOARCH     string   `json:"goarch,omitempty"`
	Trimpath   string   `json:"trimpath,omitempty"`
	SymABIs    string   `json:"symabis,omitempty"`
	Std        bool     `json:"std,omitempty"`
	NoComplete bool     `json:"noComplete,omitempty"`
	Asmhdr     string   `json:"asmhdr,omitempty"`
	GenSymABIs bool     `json:"gensymabis,omitempty"`
	Includes   []string `json:"includes,omitempty"`
}
