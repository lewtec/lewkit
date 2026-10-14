package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/build/sign"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

var (
	ErrBuildNoTarget            = errs.Wrap(errs.ErrNotFound, "build: no target")
	ErrHasNoAppPackage          = errs.Wrap(errs.ErrNotFound, "has no app package")
	ErrHostWorkflowIcons        = errors.New("icons")
	ErrIsNotADirectory          = errs.Wrap(errs.ErrInvalid, "is not a directory")
	ErrJobArchiveTargetBuild    = errors.New("build")
	ErrJobArchiveTargetSign     = errors.New("sign")
	ErrJobArchiveTargetSign2    = errors.New("sign")
	ErrUberAssembleIcons        = errors.New("icons")
	ErrUnsupportedAndroidGoarch = errs.Wrap(errs.ErrUnsupported, "unsupported android GOARCH")
)

// Uber is every archive and host package this process can finish.
// Archives are the desktop matrix. Host packages are Linux and Windows,
// a macOS or iOS app when Xcode is present, and an Android APK when an
// SDK is already installed. Out is the directory that receives them.
type Uber struct {
	Spec
	Out  string
	Name string
	SDK  string
	CGO  bool
	Sign *sign.Identity
}

// HostTargets is the host packages this process can finish without cgo.
// Linux and Windows are pure Go. macOS and iOS need xcodebuild and
// xcodegen. Android needs an SDK that is already installed.
// The cgo-free Android package is arm64. android/amd64 needs the NDK.
func HostTargets(ctx context.Context) ([]Target, error) {
	return hostTargets(ctx, false)
}

func hostTargets(ctx context.Context, cgo bool) ([]Target, error) {
	var out []Target
	for _, goarch := range []string{"amd64", "arm64"} {
		out = append(out,
			Target{GOOS: "linux", GOARCH: goarch},
			Target{GOOS: "windows", GOARCH: goarch},
		)
	}
	if darwinTools(ctx) {
		for _, goarch := range []string{"amd64", "arm64"} {
			out = append(out, Target{GOOS: "darwin", GOARCH: goarch})
		}
		out = append(out, Target{GOOS: "ios", GOARCH: "arm64"})
	}
	if apk.SDKInstalled(ctx) {
		out = append(out, Target{GOOS: "android", GOARCH: "arm64"})
		if cgo {
			out = append(out, Target{GOOS: "android", GOARCH: "amd64"})
		}
	}
	return out, nil
}

func darwinTools(ctx context.Context) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if _, err := execdriver.Which(ctx, "xcodebuild"); err != nil {
		return false
	}
	_, err := execdriver.Which(ctx, "xcodegen")
	return err == nil
}

// Run writes every archive and host package under Out.
// The steps share one workflow. Run waits on the future for that graph.
func (u Uber) Run(ctx context.Context) ([]string, error) {
	apps, err := hostTargets(ctx, u.CGO)
	if err != nil {
		return nil, err
	}
	graph, paths, iconDir, err := u.assemble(DesktopTargets(), apps)
	if err != nil {
		return nil, err
	}
	defer func() {
		if iconDir != nil && *iconDir != "" {
			os.RemoveAll(*iconDir)
		}
	}()
	fut, err := workflow.Run(ctx, graph, nil)
	if err != nil {
		return nil, err
	}
	if err := fut.Wait(ctx); err != nil {
		return nil, err
	}
	return paths, nil
}

func (u Uber) assemble(archives, apps []Target) (workflow.Graph, []string, *string, error) {
	if len(archives) == 0 && len(apps) == 0 {
		return workflow.Graph{}, nil, nil, ErrBuildNoTarget
	}
	cfg, base, err := u.Spec.Load()
	if err != nil {
		return workflow.Graph{}, nil, nil, err
	}
	id := cfg.PackageID
	if id == "" {
		id, err = release.AppID()
		if err != nil {
			return workflow.Graph{}, nil, nil, err
		}
	}
	if err := release.ValidateAppID(id); err != nil {
		return workflow.Graph{}, nil, nil, err
	}
	out := u.Out
	if out == "" {
		out = "dist"
	}
	info, err := os.Stat(out)
	if err == nil && !info.IsDir() {
		return workflow.Graph{}, nil, nil, fmt.Errorf("build: %s %w", out, ErrIsNotADirectory)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return workflow.Graph{}, nil, nil, err
	}
	name := u.Name
	if name == "" {
		name = cfg.AppName
	}
	var steps []workflow.Step
	var defaults []string
	var paths []string
	if len(archives) > 0 {
		job := Job{
			Dir:     cfg.GoMain,
			Out:     out,
			Name:    name,
			AppID:   id,
			Version: cfg.VersionName,
			Targets: archives,
			Sign:    u.Sign,
		}
		g := job.graph()
		steps = append(steps, g.Steps...)
		defaults = append(defaults, g.Defaults...)
		paths = append(paths, job.Paths()...)
	}
	var iconDir *string
	var iconDeps []string
	if len(apps) > 0 {
		iconDir = new(string)
		iconDeps = []string{"icons"}
		source := icons.MasterPath(base, cfg.Icon)
		steps = append(steps, workflow.Step{
			Name: "icons",
			Desc: "icons",
			Tasks: []workflow.Task{
				workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
					if err := ctx.Err(); err != nil {
						return err
					}
					dir, err := os.MkdirTemp("", release.Name()+"-icons-")
					if err != nil {
						return err
					}
					st.Update(dir)
					if _, err := icons.Generate(icons.Options{
						OutputDir:  dir,
						Force:      true,
						SourcePath: source,
					}); err != nil {
						os.RemoveAll(dir)
						return fmt.Errorf("%w: %w", ErrUberAssembleIcons, err)
					}
					*iconDir = dir
					return nil
				}),
			},
		})
	}
	product := common.ProductName(cfg.PackageID, cfg.AppName)
	host := Host{
		Spec: u.Spec,
		SDK:  u.SDK,
		CGO:  u.CGO,
		Sign: u.Sign,
	}
	for _, target := range apps {
		goos, goarch := target.GOOS, target.GOARCH
		dest := filepath.Join(out, AppFile(product, cfg.PackageID, goos, goarch))
		paths = append(paths, dest)
		h := host
		h.Out = dest
		h.GOARCH = goarch
		var work string
		appDeps := iconDeps
		if scaffolded(goos) {
			scaffoldName := "scaffold/" + goos + "/" + goarch
			steps = append(steps, workflow.Step{
				Name: scaffoldName,
				Deps: iconDeps,
				Desc: "scaffold " + goos + "/" + goarch,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						if err := ctx.Err(); err != nil {
							return err
						}
						dir, err := os.MkdirTemp("", release.Name()+"-host-")
						if err != nil {
							return err
						}
						if err := h.scaffold(ctx, goos, dir); err != nil {
							os.RemoveAll(dir)
							return err
						}
						work = dir
						st.Update(dir)
						return nil
					}),
				},
			})
			appDeps = []string{scaffoldName}
		}
		appName := "app/" + goos + "/" + goarch
		defaults = append(defaults, appName)
		steps = append(steps, workflow.Step{
			Name:         appName,
			Deps:         appDeps,
			Desc:         "build " + appName,
			Orchestrator: true,
			Tasks: []workflow.Task{
				workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
					if err := ctx.Err(); err != nil {
						return err
					}
					ephemeral := work != ""
					var runErr error
					if ephemeral {
						defer func() {
							if runErr == nil {
								os.RemoveAll(work)
							}
						}()
					}
					built := h
					if iconDir != nil && *iconDir != "" {
						built.IconRoot = *iconDir
					}
					if work != "" {
						built.Work = work
						built.prepared = true
					}
					path, runErr := built.packageApp(ctx, goos)
					if runErr != nil {
						return runErr
					}
					st.Update(path)
					return nil
				}),
			},
		})
	}
	return workflow.Graph{Dir: out, Steps: steps, Defaults: defaults}, paths, iconDir, nil
}
