// Package opener launches a file or URL with the host opener.
//
//	err := opener.Open(ctx, "https://lew.tec.br")
//
// Backends are xdg-open, open, and cmd /c start.
package opener

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Driver starts one target and does not wait for the launched app to exit.
type Driver interface {
	Open(ctx context.Context, target string) error
}

// Open launches target.
func Open(ctx context.Context, target string) error {
	return driver.With(ctx, func(d Driver) error {
		return d.Open(ctx, target)
	})
}
