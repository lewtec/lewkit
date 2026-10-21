package lucide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestVersionMatchesModotPin(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	cueVersion := cueLucideVersion(t, root)
	require.Equal(t, cueVersion, Version)
	require.Equal(t, cueVersion, lockLucideVersion(t, root))
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		info, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil && !info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir)
		dir = parent
	}
}

func cueLucideVersion(t *testing.T, root string) string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(root, "modot.cue"))
	require.NoError(t, err)
	value := cuecontext.New().CompileBytes(text, cue.Filename("modot.cue"))
	require.NoError(t, value.Err())
	version, err := value.LookupPath(cue.ParsePath("inputs.lucide.version")).String()
	require.NoError(t, err)
	require.NotEmpty(t, version)
	return version
}

func lockLucideVersion(t *testing.T, root string) string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(root, "modot.lock.json"))
	require.NoError(t, err)
	var lock struct {
		Dependencies []struct {
			Kind         string `json:"kind"`
			Ref          string `json:"ref"`
			CurrentValue string `json:"currentValue"`
		} `json:"dependencies"`
	}
	require.NoError(t, json.Unmarshal(text, &lock))
	var got string
	var found bool
	for _, dep := range lock.Dependencies {
		if dep.Kind == "source" && dep.Ref == "github:lucide-icons/lucide" {
			got = dep.CurrentValue
			found = true
			break
		}
	}
	require.True(t, found, "modot.lock.json lucide input")
	require.NotEmpty(t, got)
	return got
}
