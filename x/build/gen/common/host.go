package common

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/icons"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Writers fills a missing stdout from ctx and a missing stderr from stdout.
func Writers(ctx context.Context, stdout, stderr io.Writer) (io.Writer, io.Writer) {
	if stdout == nil {
		stdout = taskgroup.LineWriterFrom(ctx)
	}
	if stderr == nil {
		stderr = stdout
	}
	return stdout, stderr
}

// WorkingDir returns baseDir, or the process working directory when it is blank.
func WorkingDir(baseDir string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir != "" {
		return baseDir, nil
	}
	return os.Getwd()
}

// CleanupWorkDir removes an ephemeral work dir after a successful full build.
// A failure, KeepWorkDir, or a go-only build leaves the directory. A remove
// failure is logged when stderr is set.
func CleanupWorkDir(ephemeral bool, failed error, keep, goOnly bool, dir string, stderr io.Writer, label string) {
	if !ephemeral || failed != nil || keep || goOnly {
		return
	}
	if err := os.RemoveAll(dir); err != nil && stderr != nil {
		slog.Warn(label+" cleanup", "err", err)
	}
}

// IconRoot returns iconRoot when it is set. Otherwise it generates icons
// under workDir/folder from the master at baseDir.
func IconRoot(baseDir, icon, iconRoot, workDir, folder string) (string, error) {
	if root := strings.TrimSpace(iconRoot); root != "" {
		return root, nil
	}
	dir := filepath.Join(workDir, folder)
	if _, err := icons.Generate(icons.Options{OutputDir: dir, Force: true, SourcePath: icons.MasterPath(baseDir, icon)}); err != nil {
		return "", fmt.Errorf("%w: %w", ErrIcons, err)
	}
	return dir, nil
}

// XcodeReady reports when xcodegen and xcodebuild are on PATH.
func XcodeReady(ctx context.Context) error {
	if _, err := execdriver.Which(ctx, "xcodegen"); err != nil {
		return fmt.Errorf("%w: %w", ErrXcodeGenNotFound, err)
	}
	if _, err := execdriver.Which(ctx, "xcodebuild"); err != nil {
		return fmt.Errorf("%w: %w", ErrXcodebuildNotFound, err)
	}
	return nil
}
