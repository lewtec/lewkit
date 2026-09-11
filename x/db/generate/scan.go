package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var known = map[string]string{
	"sqlite":   "sqlite",
	"postgres": "postgresql",
}

var nameRe = regexp.MustCompile(`(?m)^--\s*name:\s*(\w+)\s+:(\w+)`)

type engine struct {
	dir   string // sqlite, postgres
	sqlc  string // sqlite, postgresql
	files []string
	named map[string]string // name → :one/:many/…
}

func scan(root string) ([]engine, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []engine
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		sqlc, ok := known[e.Name()]
		if !ok {
			continue
		}
		eng, err := loadEngine(root, e.Name(), sqlc)
		if err != nil {
			return nil, err
		}
		out = append(out, eng)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no sqlite/ or postgres/ under %s", root)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].dir < out[j].dir })
	return out, nil
}

func loadEngine(root, name, sqlc string) (engine, error) {
	eng := engine{dir: name, sqlc: sqlc, named: map[string]string{}}
	mig := filepath.Join(root, name, "migrations")
	st, err := os.Stat(mig)
	if err != nil || !st.IsDir() {
		return eng, fmt.Errorf("%s: missing migrations/", name)
	}
	matches, err := filepath.Glob(filepath.Join(root, name, "*.sql"))
	if err != nil {
		return eng, err
	}
	sort.Strings(matches)
	for _, f := range matches {
		b, err := os.ReadFile(f)
		if err != nil {
			return eng, err
		}
		for _, m := range nameRe.FindAllStringSubmatch(string(b), -1) {
			if prev, ok := eng.named[m[1]]; ok && prev != m[2] {
				return eng, fmt.Errorf("%s: %s redeclared as :%s and :%s", name, m[1], prev, m[2])
			}
			eng.named[m[1]] = m[2]
		}
		rel, err := filepath.Rel(root, f)
		if err != nil {
			return eng, err
		}
		eng.files = append(eng.files, filepath.ToSlash(rel))
	}
	if len(eng.named) == 0 {
		return eng, fmt.Errorf("%s: no -- name: queries", name)
	}
	return eng, nil
}

func compareQueries(engines []engine) error {
	if len(engines) < 2 {
		return nil
	}
	base := engines[0]
	for _, e := range engines[1:] {
		if err := sameNames(base.dir, e.dir, base.named, e.named); err != nil {
			return err
		}
	}
	return nil
}

func sameNames(a, b string, am, bm map[string]string) error {
	for n, cmd := range am {
		got, ok := bm[n]
		if !ok {
			return fmt.Errorf("query %s is in %s but not %s", n, a, b)
		}
		if got != cmd {
			return fmt.Errorf("query %s is :%s in %s and :%s in %s", n, cmd, a, got, b)
		}
	}
	for n := range bm {
		if _, ok := am[n]; !ok {
			return fmt.Errorf("query %s is in %s but not %s", n, b, a)
		}
	}
	return nil
}
