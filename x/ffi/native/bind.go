package native

import "fmt"

// Bind looks up name in lib and registers it at fnptr.
// fnptr is a pointer to a function variable.
func Bind(lib uintptr, name string, fnptr any) error {
	if _, err := Symbol(lib, name); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	Func(lib, name, fnptr)
	return nil
}
