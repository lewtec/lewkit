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
	Once        bool
	SuggestKind SuggestKind
}

// NDFA is the token automaton compiled from a command spec.
type NDFA struct {
	Start  int
	States [][]Edge
}

// CursorSet is the live NDFA configurations during a walk.
type CursorSet []cursor

type cursor struct {
	state int
	used  usedKeys
}

type usedKeys []string

func (u usedKeys) has(key string) bool {
	_, ok := slices.BinarySearch(u, key)
	return ok
}

func (u usedKeys) with(keys ...string) usedKeys {
	next := slices.Clone(u)
	for _, key := range keys {
		if key == "" || slices.Contains(next, key) {
			continue
		}
		next = append(next, key)
	}
	slices.Sort(next)
	return next
}

func (e Edge) onceKeys() []string {
	if !e.Once {
		return nil
	}
	var keys []string
	if e.Long != "" {
		keys = append(keys, "--"+e.Long)
	}
	if e.Short != 0 {
		keys = append(keys, "-"+string(e.Short))
	}
	return keys
}

func (u usedKeys) blocks(e Edge) bool {
	for _, key := range e.onceKeys() {
		if u.has(key) {
			return true
		}
	}
	return false
}

func (c cursor) after(e Edge) cursor {
	return cursor{state: e.To, used: c.used.with(e.onceKeys()...)}
}

func compareCursor(a, b cursor) int {
	if a.state != b.state {
		return a.state - b.state
	}
	return slices.Compare(a.used, b.used)
}

func sameCursor(a, b cursor) bool {
	return a.state == b.state && slices.Equal(a.used, b.used)
}

func uniqueCursors(in CursorSet) CursorSet {
	if len(in) < 2 {
		return in
	}
	slices.SortFunc(in, compareCursor)
	return slices.CompactFunc(in, sameCursor)
}

// StartSet is the epsilon-closure of the start state.
func (a *NDFA) StartSet() CursorSet {
	if a == nil {
		return nil
	}
	return a.epsilonClosure(CursorSet{{state: a.Start}})
}

func (a *NDFA) epsilonClosure(set CursorSet) CursorSet {
	if a == nil {
		return nil
	}
	var out CursorSet
	seen := make(map[int][]usedKeys)
	var walk func(cursor)
	walk = func(c cursor) {
		for _, used := range seen[c.state] {
			if slices.Equal(used, c.used) {
				return
			}
		}
		seen[c.state] = append(seen[c.state], c.used)
		out = append(out, c)
		for _, e := range a.States[c.state] {
			if e.Kind == EdgeEpsilon {
				walk(cursor{state: e.To, used: c.used})
			}
		}
	}
	for _, c := range set {
		walk(c)
	}
	return uniqueCursors(out)
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

func (a *NDFA) stepShort(set CursorSet, cluster string) CursorSet {
	if cluster == "" {
		return nil
	}
	r, size := utf8.DecodeRuneInString(cluster)
	rest := cluster[size:]
	var next CursorSet
	for _, c := range set {
		for _, e := range a.States[c.state] {
			if e.Kind != EdgeFlag || e.Short != r || c.used.blocks(e) {
				continue
			}
			taken := c.after(e)
			if rest == "" {
				next = append(next, taken)
				continue
			}
			if e.HasValue {
				value := rest
				if value[0] == '=' {
					value = value[1:]
				}
				next = append(next, a.Step(a.epsilonClosure(CursorSet{taken}), value)...)
				continue
			}
			next = append(next, a.stepShort(a.epsilonClosure(CursorSet{taken}), rest)...)
		}
	}
	return next
}

// Step consumes one committed token.
func (a *NDFA) Step(set CursorSet, token string) CursorSet {
	set = a.epsilonClosure(set)
	var next CursorSet
	if name, value, ok := strings.Cut(token, "="); ok && strings.HasPrefix(token, "--") && name != "--" && name != "" {
		long := strings.TrimPrefix(name, "--")
		if long != "" {
			for _, c := range set {
				for _, e := range a.States[c.state] {
					if e.Kind == EdgeFlag && e.Long == long && e.HasValue && !c.used.blocks(e) {
						taken := c.after(e)
						next = append(next, a.Step(a.epsilonClosure(CursorSet{taken}), value)...)
					}
				}
			}
			return uniqueCursors(a.epsilonClosure(next))
		}
	}
	for _, c := range set {
		for _, e := range a.States[c.state] {
			if c.used.blocks(e) {
				continue
			}
			switch e.Kind {
			case EdgeLiteral:
				if token == e.Literal {
					next = append(next, c.after(e))
				}
			case EdgeFlag:
				if e.Long != "" && token == "--"+e.Long {
					next = append(next, c.after(e))
				}
				if e.Short != 0 && token == "-"+string(e.Short) {
					next = append(next, c.after(e))
				}
			case EdgeValue:
				if matchValue(e, token) {
					next = append(next, c.after(e))
				}
			}
		}
	}
	if len(next) == 0 && isShortCluster(token) {
		next = append(next, a.stepShort(set, token[1:])...)
	}
	return uniqueCursors(a.epsilonClosure(next))
}

// Suggest lists outgoing labels from set that match prefix.
func (a *NDFA) Suggest(set CursorSet, prefix string) []Suggestion {
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
	for _, c := range set {
		for _, e := range a.States[c.state] {
			if c.used.blocks(e) {
				continue
			}
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

func (a *NDFA) suggestFlagEquals(set CursorSet, long, value string) []Suggestion {
	seen := make(map[string]struct{})
	var out []Suggestion
	for _, c := range set {
		for _, e := range a.States[c.state] {
			if e.Kind != EdgeFlag || e.Long != long || !e.HasValue || c.used.blocks(e) {
				continue
			}
			taken := c.after(e)
			for _, suggestion := range a.Suggest(a.epsilonClosure(CursorSet{taken}), value) {
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
