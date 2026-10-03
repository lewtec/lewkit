//go:build !windows

package windows

import (
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

func readPower() (byte, byte, byte, error) {
	return 0, 0, 0, fmt.Errorf("%w: not windows", driver.ErrIncompatible)
}
