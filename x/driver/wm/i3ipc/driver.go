package i3ipc

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/wm"
)

type backend struct {
	bin string
}

func (b backend) MoveWorkspaceToOutput(ctx context.Context, workspace string, output string) error {
	return run(ctx, b.bin, fmt.Sprintf("[workspace=%q] move workspace to output %q", workspace, output))
}

func (b backend) SwitchToWorkspace(ctx context.Context, ws string, move bool) error {
	if move {
		return run(ctx, b.bin, "move", "container", "to", "workspace", ws)
	}
	return run(ctx, b.bin, "workspace", ws)
}

func (b backend) ToggleScratchpad(ctx context.Context) error {
	return run(ctx, b.bin, "scratchpad", "show")
}

func (b backend) GetOutputs(ctx context.Context) ([]wm.Output, error) {
	return wm.JSONViaCmd[[]wm.Output](ctx, b.bin, "-t", "get_outputs")
}

func (b backend) GetWorkspaces(ctx context.Context) ([]wm.Workspace, error) {
	return wm.JSONViaCmd[[]wm.Workspace](ctx, b.bin, "-t", "get_workspaces")
}

func (b backend) GetFocusedOutput(ctx context.Context) (string, *wm.Rect, error) {
	outputs, err := b.GetOutputs(ctx)
	if err != nil {
		return "", nil, err
	}

	for _, o := range outputs {
		if o.Focused {
			return o.Name, &o.Rect, nil
		}
	}

	workspaces, err := b.GetWorkspaces(ctx)
	if err != nil {
		return "", nil, err
	}

	var focusedOutputName string
	for _, w := range workspaces {
		if w.Focused {
			focusedOutputName = w.Output
			break
		}
	}

	if focusedOutputName != "" {
		for _, o := range outputs {
			if o.Name == focusedOutputName {
				return o.Name, &o.Rect, nil
			}
		}
	}

	return "", nil, wm.ErrNoFocusedOutput
}

func (b backend) GetFocusedWindowRect(ctx context.Context) (*wm.Rect, error) {
	root, err := wm.JSONViaCmd[wm.Node](ctx, b.bin, "-t", "get_tree")
	if err != nil {
		return nil, err
	}

	found := findFocusedNode(&root)
	if found != nil {
		return &found.Rect, nil
	}

	return nil, wm.ErrNoFocusedWindow
}

func findFocusedNode(node *wm.Node) *wm.Node {
	if node.Focused {
		return node
	}
	for _, n := range node.Nodes {
		if found := findFocusedNode(n); found != nil {
			return found
		}
	}
	for _, n := range node.FloatingNodes {
		if found := findFocusedNode(n); found != nil {
			return found
		}
	}
	return nil
}

func requireBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s not found", driver.ErrIncompatible, name)
	}
	return nil
}

func run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

var _ wm.Driver = backend{}
