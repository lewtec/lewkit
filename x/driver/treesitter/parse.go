package treesitter

import "fmt"

// Lookup returns wrap(raw) when the grammar reports a non-zero language.
func Lookup[L comparable](raw L, ok bool, wrap func(L) Language) (Language, bool) {
	var zero L
	if !ok || raw == zero {
		return nil, false
	}
	return wrap(raw), true
}

// Run parses source with one grammar parser.
// syntax turns a non-nil grammar tree into a Node. Leaven recovers a lexer panic around this call.
func Run[L any, G comparable, P interface {
	Delete()
	SetLanguage(L) bool
	ParseBytes([]byte) G
}](name string, lang L, source []byte, newParser func() P, syntax func(G) Node) (*Tree, error) {
	parser := newParser()
	defer parser.Delete()
	if !parser.SetLanguage(lang) {
		return nil, fmt.Errorf("%w: %s", ErrParse, name)
	}
	parsed := parser.ParseBytes(source)
	var zero G
	if parsed == zero {
		return nil, fmt.Errorf("%w: %s", ErrParse, name)
	}
	return NewTree(syntax(parsed)), nil
}
