// Command sim drives the Inbox iOS simulator.
//
// sim depends on launch, then pings the app.
// launch depends on install.
// install depends on build and boot.
// build writes the .app. boot starts a simulator.
// Those two do not wait for each other.
// ping does not depend on launch. It talks to an app that is already open.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

var (
	ErrNoAvailableIphoneSimulator = errs.Wrap(errs.ErrUnavailable, "no available iPhone simulator")
	ErrSimLewkitModuleNotFound    = errs.Wrap(errs.ErrNotFound, "sim: lewkit module not found")
)

const (
	bundle = "br.tec.lew.eletrocromo.inbox"
	scheme = "eletrocromo-inbox"
)

func main() {
	app, err := cmd.Parse[cmd.App[simCmd]](os.Args[1:]...)
	if err != nil {
		entry.Main(context.Background(), func(context.Context) error { return err })
		return
	}
	parent := taskgroup.WithLimits(context.Background(), app.Args.Apply(taskgroup.DefaultLimits()))
	entry.MainFrom(parent, app.Run)
}

// target is one simulator step.
//
//lewkit:enum
type target int

const (
	targetSim     target = iota // sim
	targetBuild                 // build
	targetBoot                  // boot
	targetInstall               // install
	targetLaunch                // launch
	targetPing                  // ping
)

type simCmd struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
	target        cmd.EnumArg[target] `default:"sim" help:"step to run"`
}

func (simCmd) Description() string {
	return "drive the Inbox iOS simulator"
}

func (c *simCmd) Run(ctx context.Context) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	fut, err := workflow.Run(ctx, simGraph(root), []string{c.target.Value().String()})
	if err != nil {
		return err
	}
	return fut.Wait(ctx)
}

// simGraph is the Inbox simulator.
// sim depends on launch, then pings.
// launch depends on install.
// install depends on build and boot.
// ping has no dependency.
func simGraph(root string) workflow.Graph {
	app := filepath.Join(root, "dist", "Inbox.app")
	return workflow.Graph{
		Dir:      root,
		Defaults: []string{"sim"},
		Steps: []workflow.Step{
			{
				Name: "build",
				Desc: "ios app",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						st.Update(app)
						return buildInbox(ctx, root, app)
					}),
				},
			},
			{
				Name: "boot",
				Desc: "boot a simulator",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						st.Update("iPhone")
						return bootSimulator(ctx)
					}),
				},
			},
			{
				Name: "install",
				Deps: []string{"build", "boot"},
				Desc: "install after build and boot",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						st.Update(app)
						return commandRun(ctx, root, "xcrun", "simctl", "install", "booted", app)
					}),
				},
			},
			{
				Name: "launch",
				Deps: []string{"install"},
				Desc: "launch after install",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						st.Update(bundle)
						// A simulator can report the launch before the process
						// is ready. The next step waits out that gap. A failed
						// launch still continues, matching the old script.
						err := commandRun(ctx, root, "xcrun", "simctl", "launch", "booted", bundle)
						if err != nil && ctx.Err() != nil {
							return ctx.Err()
						}
						return wait(ctx, 4*time.Second)
					}),
				},
			},
			{
				Name: "ping",
				Desc: "ping the open app",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						st.Update(scheme)
						return pingInbox(ctx)
					}),
				},
			},
			{
				Name: "sim",
				Deps: []string{"launch"},
				Desc: "ping after launch",
				Pool: taskgroup.IO,
				Tasks: []workflow.Task{
					workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
						st.Update(scheme)
						return pingInbox(ctx)
					}),
				},
			},
		},
	}
}

func buildInbox(ctx context.Context, root, app string) error {
	return commandRun(ctx, root, "go", "run", "./cmd/lewkit", "release", "build",
		"--goos", "ios",
		"--goarch", "arm64",
		"--app",
		"--sdk", "iphonesimulator",
		"--config", "examples/app/inbox/eletrocromo.json",
		"--out", app,
		"--workdir", filepath.Join(root, "dist", "ios-inbox"),
	)
}

func bootSimulator(ctx context.Context) error {
	booted, err := commandOutput(ctx, "", "xcrun", "simctl", "list", "devices", "booted")
	if err != nil {
		return err
	}
	if strings.Contains(booted, "iPhone") {
		return nil
	}
	list, err := commandOutput(ctx, "", "xcrun", "simctl", "list", "devices", "available")
	if err != nil {
		return err
	}
	udid := iphoneUDID(list)
	if udid == "" {
		return ErrNoAvailableIphoneSimulator
	}
	if err := commandRun(ctx, "", "xcrun", "simctl", "boot", udid); err != nil {
		return err
	}
	if err := commandRun(ctx, "", "open", "-a", "Simulator"); err != nil {
		return err
	}
	return commandRun(ctx, "", "xcrun", "simctl", "bootstatus", udid, "-b")
}

func iphoneUDID(list string) string {
	for _, line := range strings.Split(list, "\n") {
		if !strings.Contains(line, "iPhone") {
			continue
		}
		start := strings.Index(line, "(")
		end := strings.Index(line, ")")
		if start >= 0 && end > start {
			return line[start+1 : end]
		}
	}
	return ""
}

func pingInbox(ctx context.Context) error {
	if err := commandRun(ctx, "", "xcrun", "simctl", "openurl", "booted", scheme+"://from-simctl"); err != nil {
		return err
	}
	data, err := commandOutput(ctx, "", "xcrun", "simctl", "get_app_container", "booted", bundle, "data")
	if err != nil {
		return err
	}
	cache := filepath.Join(strings.TrimSpace(data), "Library", "Caches")
	inbox := filepath.Join(cache, "inbox")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		return err
	}
	note := filepath.Join(inbox, "from-sim.md")
	if err := os.WriteFile(note, []byte("# from simctl\nhello inbox\n"), 0o644); err != nil {
		return err
	}
	body, err := json.Marshal(map[string]any{"kind": "files", "paths": []string{note}})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(cache, "open.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(append(body, '\n'))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		mod := filepath.Join(dir, "go.mod")
		cfg := filepath.Join(dir, "examples", "app", "inbox", "eletrocromo.json")
		if _, err := os.Stat(mod); err == nil {
			if _, err := os.Stat(cfg); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrSimLewkitModuleNotFound
		}
		dir = parent
	}
}

func commandRun(ctx context.Context, dir, name string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out := taskgroup.LineWriterFrom(ctx)
	cmd.Stdout = out
	cmd.Stderr = out
	return execdriver.Run(ctx, cmd)
}

func commandOutput(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := execdriver.MustCommand(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := execdriver.Output(ctx, cmd)
	return string(out), err
}
