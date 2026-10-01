package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gocmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
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
		return extractZip(archive, dir)
	}
	return extractTarGz(archive, dir)
}

func extractTarGz(archive, dir string) (string, error) {
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf("empty archive %s", archive)
		}
		if err != nil {
			return "", err
		}
		dest := filepath.Join(dir, filepath.Base(header.Name))
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return "", err
		}
		out.Close()
		return dest, nil
	}
}

func extractZip(archive, dir string) (string, error) {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	if len(zr.File) == 0 {
		return "", fmt.Errorf("empty archive %s", archive)
	}
	entry := zr.File[0]
	in, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer in.Close()
	dest := filepath.Join(dir, filepath.Base(entry.Name))
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dest, nil
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
