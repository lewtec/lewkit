package parse

// lex reads one token with the default lexer. The longest match among the
// terminals this state can act on wins. A tie goes to the higher lexical
// precedence, so a keyword beats a word of the same length. An extra is
// consumed only when it is longer than that token. EOF is reported at the
// end of src even when the state has no action on it.
func (l *Language) lex(src []byte, at int, state uint32) (id SymbolID, start, end int, ok bool) {
	if at > len(src) {
		return 0, at, at, false
	}
	startAt := at
	for {
		if at >= len(src) {
			return SymEOF, at, at, true
		}
		bestID := SymbolID(0)
		bestN := 0
		bestP := -1
		for sid := SymbolID(2); int(sid) < len(l.syms); sid++ {
			s := l.syms[sid]
			if !s.terminal || s.extra || s.match == nil {
				continue
			}
			// An immediate token matches only the bytes after the previous
			// token. Trivia that was just skipped means it does not match.
			if s.immediate && at != startAt {
				continue
			}
			if _, has := l.act[cellKey(state, sid)]; !has {
				continue
			}
			n := s.match(src, at)
			if n <= 0 {
				continue
			}
			if n > bestN || (n == bestN && s.lexPrec > bestP) {
				bestID, bestN, bestP = sid, n, s.lexPrec
			}
		}
		exN := 0
		for _, sid := range l.extras {
			s := l.syms[sid]
			if s.match == nil {
				continue
			}
			n := s.match(src, at)
			if n > exN {
				exN = n
			}
		}
		if bestN > 0 && bestN >= exN {
			return bestID, at, at + bestN, true
		}
		if exN > 0 {
			at += exN
			continue
		}
		if bestN > 0 {
			return bestID, at, at + bestN, true
		}
		return 0, at, at, false
	}
}
