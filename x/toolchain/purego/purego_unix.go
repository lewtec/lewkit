//go:build unix

package purego

import "github.com/ebitengine/purego"

// Open loads a shared library with RTLD_NOW and RTLD_GLOBAL.
func Open(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}
