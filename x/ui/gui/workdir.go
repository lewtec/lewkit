package gui

import (
	"context"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/driver/window"
	"golang.org/x/term"
)

// EnsureDir returns dir when the caller already has one.
// An empty dir on a terminal is the process working directory.
// An empty dir with no terminal opens [Welcome] and returns the folder the user picks.
// The picked folder is stored with [Remember].
func EnsureDir(ctx context.Context, dir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	openView, useCwd := workDirAction(dir, term.IsTerminal(int(os.Stdin.Fd())))
	if !openView && !useCwd {
		return existingDir(dir)
	}
	if useCwd {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return existingDir(cwd)
	}
	return pickDir(ctx)
}

func workDirAction(dir string, terminal bool) (openView, useCwd bool) {
	if strings.TrimSpace(dir) != "" {
		return false, false
	}
	if terminal {
		return false, true
	}
	return true, false
}

func existingDir(dir string) (string, error) {
	cleaned, ok := cleanDir(dir)
	if !ok {
		return "", os.ErrNotExist
	}
	return cleaned, nil
}

func pickDir(ctx context.Context) (string, error) {
	dirs, err := Recent()
	if err != nil {
		dirs = nil
	}
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit", Dirs: dirs})
	err = Open(ctx, welcome, Options{
		Config: window.Config{Title: "lewkit", Width: 880, Height: 720},
	})
	if welcome.Picked() != "" {
		if saveErr := Remember(welcome.Picked()); saveErr != nil {
			return "", saveErr
		}
		return welcome.Picked(), nil
	}
	if ctx.Err() != nil {
		return "", context.Cause(ctx)
	}
	if err != nil {
		return "", err
	}
	return "", ErrCanceled
}
