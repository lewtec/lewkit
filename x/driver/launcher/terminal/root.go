package terminal

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func init() {
	launcher.RegisterAsk("terminal", 10, "Terminal (Fuzzy)", "Terminal (Stdin)", "Terminal (y/n)", func(context.Context) error {
		return driver.TerminalGate()
	}, func(context.Context) (backend, error) {
		return backend{}, nil
	})
}
