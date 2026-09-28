package build

import "fmt"

// AndroidABI is the Android ABI for a Go GOARCH.
func AndroidABI(goarch string) (string, error) {
	switch goarch {
	case "arm64":
		return "arm64-v8a", nil
	case "arm":
		return "armeabi-v7a", nil
	case "amd64":
		return "x86_64", nil
	case "386":
		return "x86", nil
	default:
		return "", fmt.Errorf("unsupported android GOARCH %q", goarch)
	}
}
