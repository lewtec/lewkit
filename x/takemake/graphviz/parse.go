package graphviz

import (
	"errors"
	"fmt"

	xparse "github.com/lewtec/lewkit/x/parse"
	syntax "github.com/lewtec/lewkit/x/parse/graphviz"
)

func parseSource(src []byte) (*File, error) {
	src = stripBOM(src)
	model, err := syntax.Language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse(src)
	if err != nil {
		return nil, syntaxParse(err)
	}
	return lowerFile(src, root)
}

func stripBOM(src []byte) []byte {
	if len(src) >= 3 && src[0] == 0xEF && src[1] == 0xBB && src[2] == 0xBF {
		return src[3:]
	}
	return src
}

func syntaxParse(err error) error {
	var pe *xparse.Error
	if errors.As(err, &pe) && pe.Line > 0 {
		return fmt.Errorf("%w: line %d", ErrSyntax, pe.Line)
	}
	return fmt.Errorf("%w", ErrSyntax)
}

func syntaxAt(n xparse.SyntaxNode[syntax.Term, syntax.Rule]) error {
	return fmt.Errorf("%w: line %d", ErrSyntax, n.Line())
}
