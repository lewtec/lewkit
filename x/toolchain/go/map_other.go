//go:build !unix && !windows

package gotool

func prepareWrite([]byte) error { return nil }

func protectRX([]byte) error {
	return ErrJitExecutableMappingIsUnavailable
}

func mapNear(int, uintptr) ([]byte, error) {
	return nil, ErrJitExecutableMappingIsUnavailable
}
