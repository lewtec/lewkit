package hyprland

import (
	"context"
	"fmt"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/wm"
)

type backend struct{}

func (backend) MoveWorkspaceToOutput(ctx context.Context, workspace string, output string) error {
	return run(ctx, "hyprctl", "dispatch", "moveworkspacetomonitor", workspace, output)
}

func (backend) SwitchToWorkspace(ctx context.Context, ws string, move bool) error {
	cmd := "workspace"
	if move {
		cmd = "movetoworkspace"
	}
	return run(ctx, "hyprctl", "dispatch", cmd, ws)
}

func (backend) ToggleScratchpad(ctx context.Context) error {
	return run(ctx, "hyprctl", "dispatch", "togglespecialworkspace")
}

func (backend) GetOutputs(ctx context.Context) ([]wm.Output, error) {
	monitors, err := wm.JSONViaCmd[[]struct {
		Name            string `json:"name"`
		Focused         bool   `json:"focused"`
		X               int    `json:"x"`
		Y               int    `json:"y"`
		Width           int    `json:"width"`
		Height          int    `json:"height"`
		ActiveWorkspace struct {
			Name string `json:"name"`
		} `json:"activeWorkspace"`
	}](ctx, "hyprctl", "monitors", "-j")
	if err != nil {
		return nil, err
	}
	var outputs []wm.Output
	for _, m := range monitors {
		outputs = append(outputs, wm.Output{
			Name:             m.Name,
			Focused:          m.Focused,
			CurrentWorkspace: m.ActiveWorkspace.Name,
			Rect:             wm.Rect{X: m.X, Y: m.Y, Width: m.Width, Height: m.Height},
		})
	}
	return outputs, nil
}

func (backend) GetWorkspaces(ctx context.Context) ([]wm.Workspace, error) {
	workspaces, err := wm.JSONViaCmd[[]struct {
		Name    string `json:"name"`
		Monitor string `json:"monitor"`
	}](ctx, "hyprctl", "workspaces", "-j")
	if err != nil {
		return nil, err
	}

	activeWS, err := wm.JSONViaCmd[struct {
		Name string `json:"name"`
	}](ctx, "hyprctl", "activeworkspace", "-j")
	if err != nil {
		return nil, err
	}

	var result []wm.Workspace
	for _, w := range workspaces {
		result = append(result, wm.Workspace{
			Name:    w.Name,
			Output:  w.Monitor,
			Focused: w.Name == activeWS.Name,
		})
	}
	return result, nil
}

func (backend) GetFocusedOutput(ctx context.Context) (string, *wm.Rect, error) {
	outputs, err := backend{}.GetOutputs(ctx)
	if err != nil {
		return "", nil, err
	}
	for _, o := range outputs {
		if o.Focused {
			return o.Name, &o.Rect, nil
		}
	}
	return "", nil, wm.ErrNoFocusedOutput
}

func (backend) GetFocusedWindowRect(ctx context.Context) (*wm.Rect, error) {
	win, err := wm.JSONViaCmd[struct {
		At   []int `json:"at"`
		Size []int `json:"size"`
	}](ctx, "hyprctl", "activewindow", "-j")
	if err != nil {
		return nil, err
	}
	if len(win.At) != 2 || len(win.Size) != 2 {
		return nil, fmt.Errorf("%w: invalid hyprland active window geometry", wm.ErrIPC)
	}
	return &wm.Rect{X: win.At[0], Y: win.At[1], Width: win.Size[0], Height: win.Size[1]}, nil
}

func requireBinary(ctx context.Context, name string) error {
	return execdriver.RequireBinary(ctx, name)
}

func run(ctx context.Context, name string, args ...string) error {
	return execdriver.Run(ctx, execdriver.MustCommand(name, args...))
}

var _ wm.Driver = backend{}
