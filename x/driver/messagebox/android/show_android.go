//go:build android

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/androidask"
)

func show(ctx context.Context, title, message string) (string, error) {
	return androidask.Call(ctx, "alert", title, message)
}
