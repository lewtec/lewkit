package golang

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCorpus round-trips Go files in this module. Take, Make, Take must
// match. Files the model rejects stay in the unsupported count.
func diffNodes(a, b reflect.Value, path string) string {
	if !a.IsValid() || !b.IsValid() {
		if a.IsValid() == b.IsValid() {
			return ""
		}
		return path + " validity"
	}
	if a.Type() != b.Type() {
		return path + " type " + a.Type().String() + " vs " + b.Type().String()
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			if a.IsNil() == b.IsNil() {
				return ""
			}
			return fmt.Sprintf("%s nil %v vs %v (%s)", path, a.IsNil(), b.IsNil(), a.Type())
		}
		return diffNodes(a.Elem(), b.Elem(), path)
	case reflect.Struct:
		var bads []string
		for i := 0; i < a.NumField(); i++ {
			name := a.Type().Field(i).Name
			if d := diffNodes(a.Field(i), b.Field(i), path+"."+name); d != "" {
				bads = append(bads, d)
			}
		}
		return strings.Join(bads, "\n")
	case reflect.Slice:
		if a.IsNil() != b.IsNil() && (a.Len() != 0 || b.Len() != 0) {
			return fmt.Sprintf("%s nil slice %v vs %v", path, a.IsNil(), b.IsNil())
		}
		if a.Len() != b.Len() {
			return fmt.Sprintf("%s len %d vs %d", path, a.Len(), b.Len())
		}
		var bads []string
		for i := 0; i < a.Len(); i++ {
			if d := diffNodes(a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i)); d != "" {
				bads = append(bads, d)
			}
		}
		return strings.Join(bads, "\n")
	default:
		if !a.CanInterface() || reflect.DeepEqual(a.Interface(), b.Interface()) {
			if a.CanInterface() {
				return ""
			}
			return path + " unreadable"
		}
		return fmt.Sprintf("%s %#v vs %#v", path, a.Interface(), b.Interface())
	}
}

func TestCorpus(t *testing.T) {
	root := moduleRoot(t)
	var (
		ok          int
		unsupported int
		parseFailed int
		mismatches  []string
		makeFailed  []string
		other       []string
		reasons     = map[string]int{}
	)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		first, err := take(src)
		if err != nil {
			switch {
			case errors.Is(err, errSyntax):
				unsupported++
				reasons[err.Error()]++
			case errors.Is(err, errParse):
				parseFailed++
			default:
				other = append(other, rel+": "+err.Error())
			}
			return nil
		}
		var buf bytes.Buffer
		if err := first.Make(&buf); err != nil {
			makeFailed = append(makeFailed, rel+": "+err.Error())
			return nil
		}
		second, err := take(buf.Bytes())
		if err != nil {
			other = append(other, rel+": reprint: "+err.Error())
			return nil
		}
		if !reflect.DeepEqual(first, second) {
			rel = filepath.ToSlash(rel)
			if len(mismatches) < 12 {
				mismatches = append(mismatches, rel+"\n"+diffNodes(reflect.ValueOf(first), reflect.ValueOf(second), "file"))
			} else {
				mismatches = append(mismatches, rel)
			}
			return nil
		}
		ok++
		return nil
	})
	require.NoError(t, err)
	t.Logf("round-tripped %d, unsupported %d, parse %d, mismatch %d, make %d, other %d",
		ok, unsupported, parseFailed, len(mismatches), len(makeFailed), len(other))
	for reason, n := range reasons {
		t.Logf("unsupported %d %s", n, reason)
	}
	limit := min(len(mismatches), 12)
	for _, item := range mismatches[:limit] {
		t.Logf("mismatch %s", item)
	}
	for _, item := range makeFailed {
		t.Logf("make %s", item)
	}
	for _, item := range other {
		t.Logf("other %s", item)
	}
	require.Empty(t, mismatches)
	require.Empty(t, makeFailed)
	require.Empty(t, other)
	require.Positive(t, ok)
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
