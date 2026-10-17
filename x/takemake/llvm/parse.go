package llvm

import "fmt"

func parse(src string) (*File, error) {
	model, err := language()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyntax, err)
	}
	root, err := model.Parse([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyntax, err)
	}
	return lower(root)
}
