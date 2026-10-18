package llvm

import (
	"fmt"

	syntax "github.com/lewtec/lewkit/x/parse/llvm"
)

func parse(src string) (*File, error) {
	model, err := syntax.Language()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyntax, err)
	}
	root, err := model.Parse([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyntax, err)
	}
	return lower(root)
}
