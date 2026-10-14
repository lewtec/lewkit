//go:build !unix && !windows

package gotool

import "errors"

var (
	ErrJitExecutableMappingIsUnavailable2 = errors.New("jit: executable mapping is unavailable")
)

func prepareWrite([]byte) error { return nil }

func protectRX([]byte) error {
	return ErrJitExecutableMappingIsUnavailable2
}

func mapNear(int, uintptr) ([]byte, error) {
	return nil, ErrJitExecutableMappingIsUnavailable2
}
