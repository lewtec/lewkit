package avfilter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type argSet struct {
	raw string
	pos []string
	kv  map[string]string
}

func parseArgs(s string) argSet {
	a := argSet{raw: s, kv: map[string]string{}}
	if s == "" {
		return a
	}
	for _, part := range splitTop(s, ':') {
		key, val, ok := strings.Cut(part, "=")
		if ok && key != "" && isKey(key) {
			a.kv[key] = val
			continue
		}
		a.pos = append(a.pos, part)
	}
	return a
}

func (a argSet) get(keys ...string) (string, bool) {
	for _, key := range keys {
		if v, ok := a.kv[key]; ok {
			return v, true
		}
	}
	return "", false
}

func (a argSet) at(i int, keys ...string) (string, bool) {
	if v, ok := a.get(keys...); ok {
		return v, true
	}
	if i >= 0 && i < len(a.pos) {
		return a.pos[i], true
	}
	return "", false
}

func isKey(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '_' && !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

type filterTok struct {
	ins  []string
	name string
	args string
	outs []string
}

func parseFilter(tok string) (filterTok, error) {
	s := strings.TrimSpace(tok)
	var item filterTok
	for strings.HasPrefix(s, "[") {
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return item, fmt.Errorf("%w: filter label", avutil.ErrOption)
		}
		item.ins = append(item.ins, s[1:end])
		s = strings.TrimSpace(s[end+1:])
	}
	for strings.HasSuffix(s, "]") {
		open := strings.LastIndexByte(s, '[')
		if open < 0 {
			return item, fmt.Errorf("%w: filter label", avutil.ErrOption)
		}
		item.outs = append([]string{s[open+1 : len(s)-1]}, item.outs...)
		s = strings.TrimSpace(s[:open])
	}
	name, args, _ := strings.Cut(s, "=")
	item.name = strings.TrimSpace(name)
	item.args = args
	if item.name == "" {
		return item, fmt.Errorf("%w: empty filter", avutil.ErrOption)
	}
	return item, nil
}

func splitTop(s string, sep byte) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if s[i] == sep && depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, s[start:])
	return out
}
