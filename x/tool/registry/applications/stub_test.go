package applications

import (
	"context"
	"github.com/lewtec/lewkit/x/tool"
)

// stubTool is a minimal implementation of tool.Tool for use in tests,
// particularly for testing wrappers like tirithTool that decorate another Tool.
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
