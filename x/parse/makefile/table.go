package makefile

import xparse "github.com/lewtec/lewkit/x/parse"

// ParseTable compiles the declaration for table_gen.go.
// Tokens without matchers need a scanner before Compile. Take installs
// the real scanner before Load, and both scanners produce the same table.
func ParseTable() ([]byte, error) {
	prepared, err := declare()
	if err != nil {
		return nil, err
	}
	err = prepared.Scanner(func(_ []byte, at int, _ func(xparse.SymbolID) bool, _ *xparse.LexState) (xparse.SymbolID, int, int, bool) {
		return 0, at, at, false
	})
	if err != nil {
		return nil, err
	}
	return xparse.TableBytes(prepared, nil)
}
