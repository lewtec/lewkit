//go:build linux

package gtk

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/filedialog/portal"
)

func (opener) Choose(ctx context.Context, req filedialog.Request) ([]string, error) {
	return portal.Choose(ctx, portal.GTK, req)
}
