//go:build !windows

package win32

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

func ask(context.Context, string, string, string, []string) (string, bool, error) {
	return "", false, fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
