// Package nativeerr maps a binding failure onto a present sentinel.
package nativeerr

import (
	"errors"

	"github.com/lewtec/lewkit/x/driver/present"
)

// Map returns err with a present sentinel when it matches lost, size, or closed.
// A nil error stays nil. Any other error is returned as it is.
func Map(err, lost, size, closed error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, lost):
		return present.ErrLost
	case errors.Is(err, size):
		return present.ErrSize
	case errors.Is(err, closed):
		return present.ErrClosed
	default:
		return err
	}
}
