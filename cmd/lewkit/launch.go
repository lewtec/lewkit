package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build/gocmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/fs/tar"
	"github.com/lewtec/lewkit/x/fs/zip"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/sops"
)

var (
	ErrBuiltLinuxApp      = errors.New("built linux app")
	ErrBuiltWindowsApp    = errors.New("built windows app")
	ErrHasNoAppToLaunch   = errs.Wrap(errs.ErrNotFound, "has no app to launch")
	ErrNegativeOffset     = errors.New("negative offset")
	ErrSopsCommandIsEmpty = sops.ErrSopsCommandIsEmpty
)

type builtProgram struct {
	goos, goarch, archive string
	args                  []string
}

var errOtherPlatform = errors.New("built program is for another machine")

func runBuilt(ctx context.Context, prog builtProgram) error {
	if prog.goos != runtime.GOOS || prog.goarch != runtime.GOARCH {
		return fmt.Errorf("%w: %s/%s (%s); this machine is %s/%s", errOtherPlatform, prog.goos, prog.goarch, prog.archive, runtime.GOOS, runtime.GOARCH)
	}
	dir, err := os.MkdirTemp("", release.Name()+"-run-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	bin, err := extractOne(prog.archive, dir)
	if err != nil {
		return err
	}
	if err := os.Chmod(bin, 0o755); err != nil {
		return err
	}
	return runForeground(ctx, bin, prog.args...)
}

// runForeground runs name on the process terminal.
// Stdin, stdout, and stderr are the process files. A nil stderr is the taskgroup line writer.
// The watched context ignores cancel so interrupt stays with the child, which shares the process group.
func runForeground(ctx context.Context, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return execdriver.Run(context.WithoutCancel(ctx), foregroundCommand(ctx, name, args...))
}

func foregroundCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := execdriver.MustCommand(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func extractOne(archive, dir string) (string, error) {
	if strings.HasSuffix(archive, ".zip") {
		return zip.ExtractFirst(archive, dir)
	}
	return tar.ExtractFirst(archive, dir)
}

func launchApp(ctx context.Context, goos, path, id string) error {
	switch goos {
	case "android":
		adb, err := adbBin(ctx)
		if err != nil {
			return err
		}
		if err := runTool(ctx, adb, "install", "-r", path); err != nil {
			return err
		}
		return runTool(ctx, adb, "shell", "am", "start", "-n", androidComponent(id))
	case "darwin":
		return runTool(ctx, "open", path)
	case "ios":
		if err := runTool(ctx, "xcrun", "simctl", "install", "booted", path); err != nil {
			return err
		}
		return runTool(ctx, "xcrun", "simctl", "launch", "--console", "booted", id)
	case "windows":
		if runtime.GOOS != "windows" {
			return fmt.Errorf("%w (%s); this machine is %s", ErrBuiltWindowsApp, path, runtime.GOOS)
		}
		return runForeground(ctx, path)
	case "linux":
		if runtime.GOOS != "linux" {
			return fmt.Errorf("%w (%s); this machine is %s", ErrBuiltLinuxApp, path, runtime.GOOS)
		}
		return runForeground(ctx, path)
	default:
		return fmt.Errorf("%s %w", goos, ErrHasNoAppToLaunch)
	}
}

// androidComponent is the activity `am start` launches.
// The packed host dex is package lewkit. A leading dot names
// {id}.MainActivity, which is not in that dex.
func androidComponent(id string) string {
	return id + "/lewkit.MainActivity"
}

// adbBin is the adb on PATH. The Android SDK is not consulted.
func adbBin(ctx context.Context) (string, error) {
	return execdriver.Find(ctx, "adb")
}

func runTool(ctx context.Context, name string, args ...string) error {
	return gocmd.Tool(ctx, name, "", nil, args...)
}
