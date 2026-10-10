package tirith

import (
	"context"

	"github.com/lewtec/lewkit/x/tool"
)

// stubTool is a minimal tool.Tool for wrapper tests.
type stubTool struct {
	versions []string
}

func (t stubTool) ListVersions(context.Context) ([]string, error) {
	return append([]string(nil), t.versions...), nil
}

func (t stubTool) Install(context.Context, string, string) error {
	return nil
}

func (t stubTool) Pin() tool.Pin {
	var pin tool.Pin
	return pin
}
