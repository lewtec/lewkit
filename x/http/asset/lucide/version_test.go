package lucide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionMatchesModotPin(t *testing.T) {
	t.Parallel()
	require.Equal(t, lockLucideVersion(t, moduleRoot(t)), Version)
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
