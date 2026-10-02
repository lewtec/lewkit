package gui

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/driver/window"
	"golang.org/x/term"
)

// ErrNeedWindow means an empty directory with no terminal needs [Pick].
var ErrNeedWindow = errors.New("directory picker needs a window")

// EnsureDir returns dir when the caller already has one.
// An empty dir on a terminal is the process working directory.
// An empty dir with no terminal returns [ErrNeedWindow]. The caller opens a
// window and calls [Pick]. The picked folder is stored with [Remember].
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
	return "", ErrNeedWindow
}

// Pick runs [Welcome] on host until the user chooses a directory or closes the window.
func Pick(ctx context.Context, host window.Window) (string, error) {
	if host == nil {
		return "", window.ErrClosed
	}
	dirs, err := Recent()
	if err != nil {
		dirs = nil
	}
	welcome := NewWelcome(WelcomeArgs{Title: "lewkit", Dirs: dirs})
	err = Run(ctx, host, nil, welcome)
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
