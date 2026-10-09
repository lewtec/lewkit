package build

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gen/ios"
	"github.com/lewtec/lewkit/x/build/gen/mac"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

// Build runs Workflow and returns the artifact path.
// GoOnly returns the work directory. The wait is the future Run
// returns, so the path exists when Build returns.
func (host Host) Build(ctx context.Context, goos string) (string, error) {
	if ctx == nil {
		return "", ErrNilContext
	}
	var path string
	graph, err := host.workflow(goos, func(built string) { path = built })
	if err != nil {
		return "", err
	}
	fut, err := workflow.Run(ctx, graph, nil)
	if err != nil {
		return "", err
	}
	if err := fut.Wait(ctx); err != nil {
		return "", err
	}
	return path, nil
}

// Workflow is the graph release build runs for one host app.
// The icons step writes the icon tree. The scaffold step writes the
// Android, macOS, or iOS host tree. The goos/goarch step compiles,
// packages, and signs. A set IconRoot skips the icons step.
// Build waits on the future Run returns for this graph.
func (host Host) Workflow(goos string) (workflow.Graph, error) {
	return host.workflow(goos, nil)
}

func (host Host) workflow(goos string, sink func(string)) (workflow.Graph, error) {
	switch goos {
	case "darwin", "android", "ios", "windows", "linux":
	default:
		return workflow.Graph{}, fmt.Errorf("%s has no app package", goos)
	}
	cfg, base, err := host.Load()
	if err != nil {
		return workflow.Graph{}, err
	}
	source := icons.MasterPath(base, cfg.Icon)
	name := goos + "/" + host.GOARCH
	var iconDir string
	var workDir string
	var steps []workflow.Step
	var deps []string
	if host.IconRoot == "" {
		deps = []string{"icons"}
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
						return fmt.Errorf("icons: %w", err)
					}
					iconDir = dir
					return nil
				}),
			},
		})
	}
	if scaffolded(goos) {
		steps = append(steps, workflow.Step{
			Name: "scaffold",
			Deps: deps,
			Desc: "scaffold",
			Tasks: []workflow.Task{
				workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
					if err := ctx.Err(); err != nil {
						return err
					}
					dir := host.Work
					if dir == "" {
						var err error
						dir, err = os.MkdirTemp("", release.Name()+"-host-")
						if err != nil {
							return err
						}
					}
					if err := host.scaffold(ctx, goos, dir); err != nil {
						if host.Work == "" {
							os.RemoveAll(dir)
						}
						if iconDir != "" {
							os.RemoveAll(iconDir)
						}
						return err
					}
					workDir = dir
					st.Update(dir)
					return nil
				}),
			},
		})
		deps = []string{"scaffold"}
	}
	steps = append(steps, workflow.Step{
		Name: name,
		Deps: deps,
		Desc: "build " + name,
		Tasks: []workflow.Task{
			workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
				if iconDir != "" {
					defer os.RemoveAll(iconDir)
				}
				ephemeral := workDir != "" && host.Work == ""
				var runErr error
				if ephemeral {
					defer func() {
						if runErr == nil && !host.GoOnly {
							os.RemoveAll(workDir)
						}
					}()
				}
				built := host
				if iconDir != "" {
					built.IconRoot = iconDir
				}
				if workDir != "" {
					built.Work = workDir
					built.prepared = true
				}
				path, runErr := built.packageApp(ctx, goos)
				if runErr != nil {
					return runErr
				}
				if sink != nil {
					sink(path)
				}
				st.Update(path)
				return nil
			}),
		},
	})
	return workflow.Graph{Defaults: []string{name}, Steps: steps}, nil
}

func scaffolded(goos string) bool {
	switch goos {
	case "darwin", "android", "ios":
		return true
	default:
		return false
	}
}

func (host Host) scaffold(ctx context.Context, goos, workDir string) error {
	cfg, base, err := host.Load()
	if err != nil {
		return err
	}
	goMain, err := common.ResolveGoMain(cfg.GoMain, base)
	if err != nil {
		return err
	}
	switch goos {
	case "darwin":
		gen := macConfig(cfg)
		gen.GoMain = goMain
		return mac.Create(ctx, mac.Options{OutDir: workDir, Force: true, Config: gen})
	case "ios":
		gen := iosConfig(cfg)
		gen.GoMain = goMain
		return ios.Create(ctx, ios.Options{OutDir: workDir, Force: true, Config: gen})
	case "android":
		cfg.GoMain = goMain
		return apk.Create(ctx, apk.Options{OutDir: workDir, Force: true, Config: cfg})
	default:
		return nil
	}
}

func (host Host) packageApp(ctx context.Context, goos string) (string, error) {
	switch goos {
	case "darwin":
		return host.Mac(ctx)
	case "android":
		return host.Android(ctx)
	case "ios":
		return host.IOS(ctx)
	case "windows":
		return host.Windows(ctx)
	case "linux":
		return host.Linux(ctx)
	default:
		return "", fmt.Errorf("%s has no app package", goos)
	}
}
