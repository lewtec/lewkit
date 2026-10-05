//go:build !windows

package entry

func prepareHost() {}

func windowsGUI() bool { return false }
