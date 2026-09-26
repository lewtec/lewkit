// Package wm switches workspaces and reads focused geometry.
//
//	err := wm.SwitchToWorkspace(ctx, "1", false)
//	err = wm.RotateWorkspaces(ctx)
//	name, rect, err := wm.GetFocusedOutput(ctx)
//
// Import hyprland or i3ipc. Hyprland uses hyprctl. Sway uses swaymsg.
// i3 uses i3-msg.
package wm

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	// ErrIPC means a compositor command failed or returned invalid JSON.
	ErrIPC = errors.New("ipc communication failure")
	// ErrNoFocusedOutput means no output is focused.
	ErrNoFocusedOutput = errors.New("no focused output found")
	// ErrNoFocusedWindow means no window is focused.
	ErrNoFocusedWindow = errors.New("no focused window found")
)

// Rect is a geometry rectangle in pixels.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Workspace is one compositor workspace.
type Workspace struct {
	Name    string `json:"name"`
	Focused bool   `json:"focused"`
	Output  string `json:"output"`
}

// Output is one display.
type Output struct {
	Name             string `json:"name"`
	CurrentWorkspace string `json:"current_workspace"`
	Rect             Rect   `json:"rect"`
	Focused          bool   `json:"focused"`
}

// Node is one node in a Sway or i3 tree.
type Node struct {
	Rect          Rect    `json:"rect"`
	Focused       bool    `json:"focused"`
	Nodes         []*Node `json:"nodes"`
	FloatingNodes []*Node `json:"floating_nodes"`
}

// Driver controls the window manager.
type Driver interface {
	SwitchToWorkspace(ctx context.Context, ws string, move bool) error
	ToggleScratchpad(ctx context.Context) error
	GetFocusedOutput(ctx context.Context) (string, *Rect, error)
	GetFocusedWindowRect(ctx context.Context) (*Rect, error)
	GetOutputs(ctx context.Context) ([]Output, error)
	GetWorkspaces(ctx context.Context) ([]Workspace, error)
	MoveWorkspaceToOutput(ctx context.Context, workspace string, output string) error
}

// GetFocusedOutput returns the focused output name and geometry.
func GetFocusedOutput(ctx context.Context) (string, *Rect, error) {
	got, err := driver.WithResult(ctx, func(source Driver) (focused, error) {
		name, rect, err := source.GetFocusedOutput(ctx)
		return focused{name, rect}, err
	})
	return got.name, got.rect, err
}

// GetFocusedWindowRect returns the focused window geometry.
func GetFocusedWindowRect(ctx context.Context) (*Rect, error) {
	return driver.WithResult(ctx, func(source Driver) (*Rect, error) {
		return source.GetFocusedWindowRect(ctx)
	})
}

// GetOutputs lists displays.
func GetOutputs(ctx context.Context) ([]Output, error) {
	return driver.WithResult(ctx, func(source Driver) ([]Output, error) {
		return source.GetOutputs(ctx)
	})
}

// GetWorkspaces lists workspaces.
func GetWorkspaces(ctx context.Context) ([]Workspace, error) {
	return driver.WithResult(ctx, func(source Driver) ([]Workspace, error) {
		return source.GetWorkspaces(ctx)
	})
}

// MoveWorkspaceToOutput moves workspace onto output.
func MoveWorkspaceToOutput(ctx context.Context, workspace string, output string) error {
	return driver.With(ctx, func(source Driver) error {
		return source.MoveWorkspaceToOutput(ctx, workspace, output)
	})
}

type focused struct {
	name string
	rect *Rect
}
