package golang

import (
	"bytes"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/parse"
)

func TestCorpus(t *testing.T) {
	root := moduleRoot(t)
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, files)

	lang, err := Language()
	require.NoError(t, err)
	t.Logf("states %d forks %d files %d", lang.States(), lang.Forks(), len(files))

	var (
		ok      int
		badBoth int
		fails   []string
	)
	start := time.Now()
	for _, path := range files {
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		rel, _ := filepath.Rel(root, path)
		_, perr := parser.ParseFile(token.NewFileSet(), path, src, parser.SkipObjectResolution)
		t0 := time.Now()
		tree, oerr := Parse(src)
		if d := time.Since(t0); d > 2*time.Second {
			t.Logf("slow %s %d bytes %s", rel, len(src), d.Round(time.Millisecond))
		}
		switch {
		case perr != nil && oerr != nil:
			require.ErrorIs(t, oerr, errs.ErrSyntax)
			badBoth++
		case perr != nil:
			fails = append(fails, rel+" accepted, go/parser rejected: "+firstLine(perr.Error()))
		case oerr != nil:
			fails = append(fails, rel+" rejected: "+firstLine(oerr.Error()))
		default:
			if msg := diffTokens(src, tree.Leaves()); msg != "" {
				fails = append(fails, rel+" tokens: "+msg)
			} else {
				ok++
			}
		}
		if len(fails) > 8 {
			break
		}
		if ok > 0 && ok%400 == 0 {
			t.Logf("%d files in %s, last %s (%d bytes)", ok, time.Since(start).Round(time.Millisecond), rel, len(src))
		}
	}
	t.Logf("accepted %d, both rejected %d, failures %d, %s", ok, badBoth, len(fails), time.Since(start).Round(time.Millisecond))
	for _, f := range fails {
		t.Log(f)
	}
	require.Empty(t, fails)
	require.Positive(t, ok)
}

func diffTokens(src []byte, leaves []parse.Leaf) string {
	want := scanAll(src)
	var got []parse.Leaf
	for _, leaf := range leaves {
		if leaf.Name == "eof" {
			continue
		}
		got = append(got, leaf)
	}
	n := len(want)
	if len(got) < n {
		n = len(got)
	}
	for i := 0; i < n; i++ {
		if want[i].kind != got[i].Name {
			return mismatch(i, want[i], got[i])
		}
		if want[i].lit == "" {
			continue
		}
		text := got[i].Text
		if want[i].kind == "STRING" {
			text = string(bytes.ReplaceAll([]byte(text), []byte{'\r'}, nil))
		}
		if text != want[i].lit {
			return mismatch(i, want[i], parse.Leaf{Name: got[i].Name, Text: text, Start: got[i].Start, End: got[i].End})
		}
	}
	if len(want) != len(got) {
		return "count scanner " + itoa(len(want)) + " leaves " + itoa(len(got))
	}
	return ""
}

func scanAll(src []byte) []scanned {
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var bad bool
	s.Init(file, src, func(token.Position, string) { bad = true }, 0)
	var out []scanned
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.ILLEGAL || bad {
			return nil
		}
		out = append(out, scanned{kind: tok.String(), lit: lit})
	}
	return out
}

func mismatch(i int, want scanned, got parse.Leaf) string {
	return "at " + itoa(i) + " scanner " + want.kind + " " + quote(want.lit) + " leaf " + got.Name + " " + quote(got.Text)
}

func quote(s string) string {
	if len(s) > 40 {
		s = s[:40]
	}
	return strconvQuote(s)
}

func strconvQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [16]byte
	i := len(d)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		d[i] = '-'
	}
	return string(d[i:])
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 180 {
		s = s[:180]
	}
	return s
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir)
		dir = parent
	}
}
