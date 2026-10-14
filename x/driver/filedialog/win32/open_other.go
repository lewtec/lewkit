//go:build !windows

package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

func (opener) Choose(context.Context, filedialog.Request) ([]string, error) {
	return nil, driver.ForGOOS("windows")
}
