//go:build !android

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

func show(context.Context, string, string) (string, error) {
	return "", fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
