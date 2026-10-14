//go:build !windows

package windows

import (
	"github.com/lewtec/lewkit/x/driver"
)

func readPower() (byte, byte, byte, error) {
	return 0, 0, 0, driver.ForGOOS("windows")
}
