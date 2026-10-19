package c

import (
	"fmt"

	syntax "github.com/lewtec/lewkit/x/parse/c"
)

func parseSource(src []byte) (*File, error) {
	src = splice(src)
	model, err := syntax.Language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errSyntax, err)
	}
	return lower(src, root)
}
