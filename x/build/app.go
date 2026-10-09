package build

import (
	"context"
	"fmt"
	"os"

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
// The icons step writes the icon tree the host packages already apply.
// The goos/goarch step compiles, packages, and signs that app.
// A set IconRoot skips the icons step and uses that tree.
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
	steps = append(steps, workflow.Step{
		Name: name,
		Deps: deps,
		Desc: "build " + name,
		Tasks: []workflow.Task{
			workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
				if iconDir != "" {
					defer os.RemoveAll(iconDir)
				}
				built := host
				if iconDir != "" {
					built.IconRoot = iconDir
				}
				path, err := built.packageApp(ctx, goos)
				if err != nil {
					return err
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
