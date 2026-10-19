package uikitapp

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
)

// OwnsUI reports whether this process presents UIKit.
// ELETROCROMO_NO_UI and LEWKIT_NO_UI mean a host owns the window.
func OwnsUI() bool {
	if runtime.GOOS != "ios" {
		return false
	}
	if driver.EnvFlag("ELETROCROMO_NO_UI") || driver.EnvFlag("LEWKIT_NO_UI") {
		return false
	}
	return true
}

// RequireProcess reports ErrIncompatible unless this process presents UIKit.
func RequireProcess() error {
	if err := driver.ForGOOS(build.GOOSIOS); err != nil {
		return err
	}
	if OwnsUI() {
		return nil
	}
	return fmt.Errorf("%w: hosted", driver.ErrIncompatible)
}

// Accepted maps an askwire reply onto nil or ErrUnavailable.
func Accepted(raw string) error {
	status, payload := askwire.Split(raw)
	if status == askwire.StatusOK {
		return nil
	}
	msg := strings.TrimSpace(payload)
	if msg == "" {
		msg = string(status)
	}
	return fmt.Errorf("%w: %s", driver.ErrUnavailable, msg)
}
