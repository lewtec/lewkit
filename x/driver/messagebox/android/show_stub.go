//go:build !android

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

func show(context.Context, string, string) (string, error) {
	return "", driver.JavaVM()
}
