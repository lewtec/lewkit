package llvm

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeavenCorpus(t *testing.T) {
	root := filepath.Join("testdata", "corpus")
	info, err := os.Stat(root)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	var fail []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ll") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		first, err := Take(strings.NewReader(string(raw)))
		if err != nil {
			fail = append(fail, rel+": take: "+err.Error())
			return nil
		}
		text := new(strings.Builder)
		if err := first.Make(text); err != nil {
			fail = append(fail, rel+": make: "+err.Error())
			return nil
		}
		second, err := Take(strings.NewReader(text.String()))
		if err != nil {
			fail = append(fail, rel+": retake: "+err.Error())
			return nil
		}
		again := new(strings.Builder)
		if err := second.Make(again); err != nil {
			fail = append(fail, rel+": make2: "+err.Error())
			return nil
		}
		if text.String() != again.String() {
			fail = append(fail, rel+": print drift")
			return nil
		}
		if !reflect.DeepEqual(first, second) {
			fail = append(fail, rel+": model drift")
		}
		return nil
	})
	require.NoError(t, err)
	if len(fail) > 0 {
		t.Fatalf("%d failures\n%s", len(fail), strings.Join(fail, "\n"))
	}
}
