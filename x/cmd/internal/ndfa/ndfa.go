// Package ndfa is the token automaton behind command completion and graphs.
package ndfa

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// EdgeKind is the label class on an NDFA transition.
type EdgeKind byte

const (
	EdgeEpsilon EdgeKind = iota
	EdgeLiteral
	EdgeFlag
	EdgeValue
)

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

// Edge is one NDFA transition.
type Edge struct {
	Kind        EdgeKind
	To          int
	Literal     string
	Long        string
	Short       rune
	Help        string
	Choices     []string
	AcceptAny   bool
	HasValue    bool
	SuggestKind SuggestKind
}

// NDFA is the token automaton compiled from a command spec.
type NDFA struct {
	Start  int
	States [][]Edge
}

func (a *NDFA) epsilonClosure(set []int) []int {
	if a == nil {
		return nil
	}
	seen := make([]bool, len(a.States))
	var out []int
	var walk func(int)
	walk = func(state int) {
		if seen[state] {
			return
		}
		seen[state] = true
		out = append(out, state)
		for _, e := range a.States[state] {
			if e.Kind == EdgeEpsilon {
				walk(e.To)
			}
		}
	}
	for _, state := range set {
		walk(state)
	}
	return out
}

func uniqueStates(in []int) []int {
	if len(in) < 2 {
		return in
	}
	slices.Sort(in)
	return slices.Compact(in)
}

func isOption(token string) bool {
	return len(token) > 1 && token[0] == '-' && token != "-"
}

func matchValue(e Edge, token string) bool {
	if token == "--" {
		return false
	}
	if isOption(token) && !e.AcceptAny {
		return false
	}
	if len(e.Choices) > 0 && !slices.Contains(e.Choices, token) {
		return false
	}
	return true
}

func isShortCluster(token string) bool {
	return len(token) > 2 && token[0] == '-' && token[1] != '-'
}

func (a *NDFA) stepShort(set []int, cluster string) []int {
	if cluster == "" {
		return nil
	}
	r, size := utf8.DecodeRuneInString(cluster)
	rest := cluster[size:]
	var next []int
	for _, state := range set {
		for _, e := range a.States[state] {
			if e.Kind != EdgeFlag || e.Short != r {
				continue
			}
			if rest == "" {
				next = append(next, e.To)
				continue
			}
			if e.HasValue {
				if rest[0] == '=' {
					rest = rest[1:]
				}
				next = append(next, a.Step(a.epsilonClosure([]int{e.To}), rest)...)
				continue
			}
			next = append(next, a.stepShort(a.epsilonClosure([]int{e.To}), rest)...)
		}
	}
	return next
}

// Step consumes one committed token.
func (a *NDFA) Step(set []int, token string) []int {
	set = a.epsilonClosure(set)
	var next []int
	if name, value, ok := strings.Cut(token, "="); ok && strings.HasPrefix(token, "--") && name != "--" && name != "" {
		long := strings.TrimPrefix(name, "--")
		if long != "" {
			for _, state := range set {
				for _, e := range a.States[state] {
					if e.Kind == EdgeFlag && e.Long == long && e.HasValue {
						next = append(next, a.Step(a.epsilonClosure([]int{e.To}), value)...)
					}
				}
			}
			return uniqueStates(a.epsilonClosure(next))
		}
	}
	for _, state := range set {
		for _, e := range a.States[state] {
			switch e.Kind {
			case EdgeLiteral:
				if token == e.Literal {
					next = append(next, e.To)
				}
			case EdgeFlag:
				if e.Long != "" && token == "--"+e.Long {
					next = append(next, e.To)
				}
				if e.Short != 0 && token == "-"+string(e.Short) {
					next = append(next, e.To)
				}
			case EdgeValue:
				if matchValue(e, token) {
					next = append(next, e.To)
				}
			}
		}
	}
	if len(next) == 0 && isShortCluster(token) {
		next = append(next, a.stepShort(set, token[1:])...)
	}
	return uniqueStates(a.epsilonClosure(next))
}

// Suggest lists outgoing labels from set that match prefix.
func (a *NDFA) Suggest(set []int, prefix string) []Suggestion {
	set = a.epsilonClosure(set)
	if long, value, ok := strings.Cut(strings.TrimPrefix(prefix, "--"), "="); ok && strings.HasPrefix(prefix, "--") && long != "" {
		return a.suggestFlagEquals(set, long, value)
	}
	seen := make(map[string]struct{})
	var out []Suggestion
	add := func(s Suggestion) {
		if prefix != "" && !strings.HasPrefix(s.Text, prefix) {
			return
		}
		if _, ok := seen[s.Text]; ok {
			return
		}
		seen[s.Text] = struct{}{}
		out = append(out, s)
	}
	for _, state := range set {
		for _, e := range a.States[state] {
			switch e.Kind {
			case EdgeLiteral:
				add(Suggestion{Text: e.Literal, Help: e.Help, Kind: e.SuggestKind})
			case EdgeFlag:
				if e.Long != "" {
					add(Suggestion{Text: "--" + e.Long, Help: e.Help, Kind: SuggestFlag})
				}
				if e.Short != 0 {
					add(Suggestion{Text: "-" + string(e.Short), Help: e.Help, Kind: SuggestFlag})
				}
			case EdgeValue:
				for _, choice := range e.Choices {
					add(Suggestion{Text: choice, Help: e.Help, Kind: SuggestValue})
				}
			}
		}
	}
	slices.SortFunc(out, compareSuggestion)
	return out
}

func (a *NDFA) suggestFlagEquals(set []int, long, value string) []Suggestion {
	seen := make(map[string]struct{})
	var out []Suggestion
	for _, state := range set {
		for _, e := range a.States[state] {
			if e.Kind != EdgeFlag || e.Long != long || !e.HasValue {
				continue
			}
			for _, suggestion := range a.Suggest(a.epsilonClosure([]int{e.To}), value) {
				text := "--" + long + "=" + suggestion.Text
				if _, ok := seen[text]; ok {
					continue
				}
				seen[text] = struct{}{}
				suggestion.Text = text
				out = append(out, suggestion)
			}
		}
	}
	slices.SortFunc(out, compareSuggestion)
	return out
}

func compareSuggestion(a, b Suggestion) int {
	if a.Kind != b.Kind {
		return int(a.Kind) - int(b.Kind)
	}
	return strings.Compare(a.Text, b.Text)
}
