package header

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func bindFuncName(unit *Unit) error {
	if unit.BindName == "" {
		unit.BindName = "Symbols"
	}
	if !token.IsIdentifier(unit.BindName) || token.IsKeyword(unit.BindName) {
		return fmt.Errorf("%w: %s", errPackage, unit.BindName)
	}
	return nil
}

func assignNames(unit *Unit) {
	used := map[string]bool{
		unit.BindName: true,
		"Open":        true,
		"copyAt":      true,
	}
	for _, t := range unit.Types {
		if t.Go != "" {
			used[t.Go] = true
			continue
		}
		base := t.prefer
		if base == "" {
			base = t.Tag
		}
		if base == "" {
			base = t.Spell
		}
		if base == "" || base == "struct" || base == "union" || base == "enum" {
			base = "Anon"
		}
		t.Go = take(ExportName(base), used)
	}
	for _, t := range unit.Types {
		usedFields := map[string]bool{}
		anon := 0
		for _, f := range t.Fields {
			if f.Type != nil && f.Type.flex() {
				continue
			}
			if f.IsBit && f.BitWidth == 0 {
				continue
			}
			raw := f.Name
			if raw == "" {
				raw = "Anon" + strconv.Itoa(anon)
				anon++
			}
			f.Go = take(ExportName(raw), usedFields)
		}
		for i := range t.Enums {
			t.Enums[i].Go = take(ExportName(t.Enums[i].Name), used)
		}
	}
	for i := range unit.Consts {
		unit.Consts[i].Go = take(ExportName(unit.Consts[i].Name), used)
	}
	for i := range unit.Funcs {
		fn := &unit.Funcs[i]
		fn.Go = take(ExportName(fn.Name), used)
		seen := map[string]bool{}
		for j := range fn.Params {
			fn.Params[j].Name = paramName(fn.Params[j].Name, seen)
		}
	}
	for i := range unit.Objects {
		unit.Objects[i].Go = take(ExportName(unit.Objects[i].Name), used)
	}
}

func paramName(name string, seen map[string]bool) string {
	if name == "" {
		return ""
	}
	if !token.IsIdentifier(name) || token.IsKeyword(name) {
		name = ExportName(name)
	}
	if seen[name] {
		name += "2"
	}
	seen[name] = true
	return name
}

// ExportName is the Go identifier for a C name.
// A single token is capitalized. Snake case becomes Pascal case.
// An empty result is Anon. A leading digit gains an N prefix.
// A keyword gains an X suffix. A result that is still not an identifier is Name.
func ExportName(name string) string {
	parts := strings.Split(name, "_")
	var b strings.Builder
	if len(parts) == 1 {
		b.WriteString(capitalize(parts[0]))
	} else {
		for _, part := range parts {
			if part == "" {
				continue
			}
			b.WriteString(capitalize(part))
		}
	}
	s := b.String()
	if s == "" {
		s = "Anon"
	}
	r, _ := utf8.DecodeRuneInString(s)
	if r >= '0' && r <= '9' {
		s = "N" + s
	}
	if token.IsKeyword(s) {
		s += "X"
	}
	if !token.IsIdentifier(s) {
		s = "Name"
	}
	return s
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	if unicode.IsLower(r) {
		return string(unicode.ToUpper(r)) + s[size:]
	}
	return s
}

func take(name string, used map[string]bool) string {
	if name == "" {
		name = "Anon"
	}
	if !used[name] {
		used[name] = true
		return name
	}
	alt := name + "Func"
	if !used[alt] {
		used[alt] = true
		return alt
	}
	for i := 2; ; i++ {
		cand := alt + strconv.Itoa(i)
		if !used[cand] {
			used[cand] = true
			return cand
		}
	}
}
