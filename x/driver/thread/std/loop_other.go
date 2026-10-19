//go:build !ios

package std

import "context"

func (t *backend) loopIOS(context.Context) {}

func wakeLoop() {}
