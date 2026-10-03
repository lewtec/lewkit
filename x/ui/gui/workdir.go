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
// release run keeps stdin a terminal, so an app that must show the window calls [ChooseDir].
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

// ChooseDir opens the welcome window and returns the folder the user picks.
// A nil Dirs list loads [Recent]. The picked folder is stored with [Remember].
// Closing the window returns [ErrCanceled].
// [EnsureDir] does not call this when stdin is a terminal.
func ChooseDir(ctx context.Context, args WelcomeArgs) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	if args.Dirs == nil {
		dirs, err := Recent()
		if err != nil {
			dirs = nil
		}
		args.Dirs = dirs
	}
	welcome := NewWelcome(args)
	err := Open(ctx, welcome, Options{
		Config: window.Config{Title: welcome.title, Width: 880, Height: 720},
	})
	return finishChoose(ctx, welcome, err)
}

func pickDir(ctx context.Context) (string, error) {
	return ChooseDir(ctx, WelcomeArgs{Title: "lewkit"})
}

func finishChoose(ctx context.Context, welcome *Welcome, openErr error) (string, error) {
	if path := welcome.Picked(); path != "" {
		if err := Remember(path); err != nil {
			return "", err
		}
		return path, nil
	}
	if ctx.Err() != nil {
		return "", context.Cause(ctx)
	}
	if openErr != nil {
		return "", openErr
	}
	return "", ErrCanceled
}
