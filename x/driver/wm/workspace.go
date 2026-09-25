package wm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/media"
)

var opMu sync.Mutex

// SwitchToWorkspace shows ws. When move is set, the focused container moves with it.
func SwitchToWorkspace(ctx context.Context, ws string, move bool) error {
	opMu.Lock()
	defer opMu.Unlock()
	return switchToWorkspace(ctx, ws, move)
}

func switchToWorkspace(ctx context.Context, ws string, move bool) error {
	return driver.With(ctx, func(source Driver) error {
		return source.SwitchToWorkspace(ctx, ws, move)
	})
}

// ToggleScratchpad shows or hides the scratchpad.
func ToggleScratchpad(ctx context.Context) error {
	opMu.Lock()
	defer opMu.Unlock()
	return toggleScratchpad(ctx)
}

func toggleScratchpad(ctx context.Context) error {
	return driver.With(ctx, func(source Driver) error {
		return source.ToggleScratchpad(ctx)
	})
}

// ToggleScratchpadWithInfo toggles the scratchpad, then posts the playing track.
// A missing player does not fail the toggle.
func ToggleScratchpadWithInfo(ctx context.Context) error {
	if err := ToggleScratchpad(ctx); err != nil {
		return err
	}
	if err := media.ShowStatus(ctx); err != nil {
		slog.ErrorContext(ctx, "scratchpad media status", "error", err)
	}
	return nil
}

// NextWorkspace switches to the next numbered workspace.
// The counter is $XDG_RUNTIME_DIR/lewkit/last-workspace and starts at 10.
// When XDG_RUNTIME_DIR is unset the file is under os.TempDir()/lewkit-<uid>.
func NextWorkspace(ctx context.Context, move bool) error {
	opMu.Lock()
	defer opMu.Unlock()
	dir, err := runtimeDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "last-workspace")
	last := 10
	if data, err := os.ReadFile(path); err == nil {
		if val, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			last = val
		}
	}
	next := strconv.Itoa(last + 1)
	if err := os.WriteFile(path, []byte(next), 0o600); err != nil {
		return err
	}
	return switchToWorkspace(ctx, next, move)
}

// RotateWorkspaces moves each occupied workspace onto the next output.
// One output or fewer is a no-op. The focused workspace is shown again at the end.
func RotateWorkspaces(ctx context.Context) error {
	opMu.Lock()
	defer opMu.Unlock()
	workspaces, err := GetWorkspaces(ctx)
	if err != nil {
		return err
	}
	var focused string
	for _, w := range workspaces {
		if w.Focused {
			focused = w.Name
			break
		}
	}
	outputs, err := GetOutputs(ctx)
	if err != nil {
		return err
	}
	var screens []string
	onScreen := map[string]string{}
	for _, o := range outputs {
		if o.CurrentWorkspace == "" {
			continue
		}
		screens = append(screens, o.Name)
		onScreen[o.Name] = o.CurrentWorkspace
	}
	if len(screens) <= 1 {
		return nil
	}
	old := append([]string(nil), screens...)
	last := screens[len(screens)-1]
	screens = append([]string{last}, screens[:len(screens)-1]...)
	var errs []error
	for i, from := range old {
		if err := MoveWorkspaceToOutput(ctx, onScreen[from], screens[i]); err != nil {
			errs = append(errs, err)
		}
	}
	if focused != "" {
		if err := switchToWorkspace(ctx, focused, false); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func runtimeDir() (string, error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("lewkit-%d", os.Getuid()))
	}
	return filepath.Join(dir, "lewkit"), nil
}
