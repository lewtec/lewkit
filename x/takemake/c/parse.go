package c

import "fmt"

func parseSource(src []byte) (*File, error) {
	src = splice(src)
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errSyntax, err)
	}
	return lower(src, root)
}
