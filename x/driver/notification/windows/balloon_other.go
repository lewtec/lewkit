//go:build !windows

package windows

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

func available(context.Context) error {
	return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}

func (backend) Notify(context.Context, notification.Notification) error {
	return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
