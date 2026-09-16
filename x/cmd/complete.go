package cmd

import (
	"github.com/lewtec/lewkit/x/cmd/internal/ndfa"
	"github.com/lewtec/lewkit/x/graph"
)

type SuggestKind = ndfa.SuggestKind

const (
	SuggestCommand = ndfa.SuggestCommand
	SuggestFlag    = ndfa.SuggestFlag
	SuggestValue   = ndfa.SuggestValue
	SuggestDash    = ndfa.SuggestDash
)

// Suggestion is one token Complete may insert.
type Suggestion = ndfa.Suggestion

// Complete suggests the next token for T. args is the words after the
// program name; the last word is the prefix being completed and may be empty.
func Complete[T any](args ...string) ([]Suggestion, error) {
	automaton, err := compileNDFA[T]()
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
		set = automaton.Step(set, word)
		if len(set) == 0 {
			return nil, nil
		}
	}
	return automaton.Suggest(set, prefix), nil
}

// Graph is the token NDFA for T as a directed graph (DOT / Mermaid).
func Graph[T any]() (graph.Graph, error) {
	automaton, err := compileNDFA[T]()
	if err != nil {
		return graph.Graph{}, err
	}
	return automaton.Graph(), nil
}
