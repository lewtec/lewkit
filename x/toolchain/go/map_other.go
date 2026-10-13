//go:build !unix && !windows

package gotool

import "fmt"

func prepareWrite([]byte) error { return nil }

func protectRX([]byte) error {
	return fmt.Errorf("jit: executable mapping is unavailable")
}

func mapNear(int, uintptr) ([]byte, error) {
	return nil, fmt.Errorf("jit: executable mapping is unavailable")
}
