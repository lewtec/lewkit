package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gocmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/fs/tar"
	"github.com/lewtec/lewkit/x/fs/zip"
)

type builtProgram struct {
	goos, goarch, archive string
	args                  []string
}

func runBuilt(ctx context.Context, prog builtProgram) error {
	if prog.goos != runtime.GOOS || prog.goarch != runtime.GOARCH {
		return fmt.Errorf("built %s/%s (%s); this machine is %s/%s", prog.goos, prog.goarch, prog.archive, runtime.GOOS, runtime.GOARCH)
	}
	dir, err := os.MkdirTemp("", "lewkit-run-")
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
	return execdriver.Run(context.WithoutCancel(ctx), foregroundCommand(name, args...))
}

func foregroundCommand(name string, args ...string) *exec.Cmd {
	cmd := execdriver.MustCommand(name, args...)
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
		adb := adbBin(ctx)
		if err := runTool(ctx, adb, "install", "-r", path); err != nil {
			return err
		}
		return runTool(ctx, adb, "shell", "am", "start", "-n", id+"/.MainActivity")
	case "darwin":
		return runTool(ctx, "open", path)
	case "ios":
		if err := runTool(ctx, "xcrun", "simctl", "install", "booted", path); err != nil {
			return err
		}
		return runTool(ctx, "xcrun", "simctl", "launch", "booted", id)
	default:
		return fmt.Errorf("%s has no app to launch", goos)
	}
}

func adbBin(ctx context.Context) string {
	home, err := build.AndroidSDK(ctx)
	if err != nil {
		home = os.Getenv("ANDROID_HOME")
	}
	if home == "" {
		return "adb"
	}
	bin := filepath.Join(home, "platform-tools", "adb")
	if _, err := os.Stat(bin); err != nil {
		return "adb"
	}
	return bin
}

func runTool(ctx context.Context, name string, args ...string) error {
	return gocmd.Tool(ctx, name, "", nil, args...)
}
