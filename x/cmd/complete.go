package cmd

import "reflect"

// SuggestKind is what a Suggestion completes.
type SuggestKind int

const (
	SuggestCommand SuggestKind = iota
	SuggestFlag
	SuggestValue
	SuggestDash
)

// Suggestion is one token Complete may insert.
type Suggestion struct {
	Text string
	Help string
	Kind SuggestKind
}

// Complete suggests the next token for T. args is the words after the
// program name; the last word is the prefix being completed and may be empty.
func Complete[T any](args ...string) ([]Suggestion, error) {
	var zero T
	n, err := compileNFA(reflect.ValueOf(&zero).Elem())
	if err != nil {
		return nil, err
	}
	prefix := ""
	words := args
	if len(args) > 0 {
		prefix = args[len(args)-1]
		words = args[:len(args)-1]
	}
	set := []int{n.start}
	for _, w := range words {
		set = n.step(set, w)
		if len(set) == 0 {
			return nil, nil
		}
	}
	return n.suggest(set, prefix), nil
}
