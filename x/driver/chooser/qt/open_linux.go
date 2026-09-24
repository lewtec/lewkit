//go:build linux

package qt

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/chooser"
	"github.com/lewtec/lewkit/x/driver/chooser/portal"
)

func (opener) Choose(ctx context.Context, req chooser.Request) ([]string, error) {
	return portal.Choose(ctx, portal.KDE, req)
}
