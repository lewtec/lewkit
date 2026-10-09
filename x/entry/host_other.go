//go:build !windows && !linux

package entry

import "context"

func prepareHost(context.Context) {}

func windowsGUI() bool { return false }

func linuxApp() bool { return false }
