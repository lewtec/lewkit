package cmd

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
	automaton, err := CompileNDFA[T]()
	if err != nil {
		return nil, err
	}
	prefix := ""
	words := args
	if len(args) > 0 {
		prefix = args[len(args)-1]
		words = args[:len(args)-1]
	}
	set := []int{automaton.Start}
	for _, word := range words {
		set = automaton.step(set, word)
		if len(set) == 0 {
			return nil, nil
		}
	}
	return automaton.suggest(set, prefix), nil
}
