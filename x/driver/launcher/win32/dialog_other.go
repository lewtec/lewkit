//go:build !windows

package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

func ask(context.Context, string, string, string, []string) (string, bool, error) {
	return "", false, driver.ForGOOS("windows")
}
