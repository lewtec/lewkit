package golang

import (
	"sync"

	"github.com/lewtec/lewkit/x/parse"
)

var (
	once sync.Once
	lang *parse.Language
	lerr error
)

// Language is the compiled Go grammar.
func Language() (*parse.Language, error) {
	once.Do(func() {
		lang, lerr = define().Compile()
	})
	return lang, lerr
}

// Parse parses one Go source file. The tree's leaves are the tokens
// go/scanner would return, not counting comments.
func Parse(src []byte) (*parse.Tree, error) {
	l, err := Language()
	if err != nil {
		return nil, err
	}
	return l.Parse(src)
}
