package build

import (
	"fmt"

	"github.com/lewtec/lewkit/x/errs"
)

// ABI is an Android native library ABI.
//
//lewkit:enum
type ABI string

const (
	ABIARM64V8A   ABI = "arm64-v8a"
	ABIARMEABIV7A ABI = "armeabi-v7a"
	ABIX86_64     ABI = "x86_64"
	ABIX86        ABI = "x86"
)

// GOARCH returns the Go architecture for abi.
func (abi ABI) GOARCH() GOARCH {
	switch abi {
	case ABIARM64V8A:
		return GOARCHARM64
	case ABIARMEABIV7A:
		return GOARCHARM
	case ABIX86_64:
		return GOARCHAMD64
	case ABIX86:
		return GOARCH386
	default:
		return ""
	}
}

// ErrUnsupportedAndroidGoarch means GOARCH has no Android ABI.
var ErrUnsupportedAndroidGoarch = errs.Wrap(errs.ErrUnsupported, "unsupported android GOARCH")

// ABIFor returns the Android ABI for a Go architecture.
func ABIFor(goarch GOARCH) (ABI, error) {
	switch goarch {
	case GOARCHARM64:
		return ABIARM64V8A, nil
	case GOARCHARM:
		return ABIARMEABIV7A, nil
	case GOARCHAMD64:
		return ABIX86_64, nil
	case GOARCH386:
		return ABIX86, nil
	default:
		return "", fmt.Errorf("%w: %w %q", ErrABI, ErrUnsupportedAndroidGoarch, goarch)
	}
}

// AndroidABI is the Android ABI for a Go GOARCH.
func AndroidABI(goarch string) (string, error) {
	abi, err := ABIFor(GOARCH(goarch))
	if err != nil {
		return "", err
	}
	return string(abi), nil
}
