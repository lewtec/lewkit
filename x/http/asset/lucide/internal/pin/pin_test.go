package pin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCueInputVersion(t *testing.T) {
	t.Parallel()
	text := `
inputs: htmx: {
	from:    "github:bigskysoftware/htmx"
	version: "v4.0.0"
}
inputs: lucide: {
	from:    "github:lucide-icons/lucide"
	version: "1.57.0"
}
`
	got, err := cueInputVersion(text, "lucide")
	require.NoError(t, err)
	require.Equal(t, "1.57.0", got)
	_, err = cueInputVersion(text, "missing")
	require.ErrorIs(t, err, errCue)
}

func TestVersionAgrees(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cue := "inputs: lucide: {\n\tversion: \"9.9.9\"\n}\n"
	lock := `{
  "dependencies": [
    {
      "kind": "source",
      "ref": "github:lucide-icons/lucide",
      "currentValue": "9.9.9"
    }
  ]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.cue"), []byte(cue), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.lock.json"), []byte(lock), 0o644))
	got, err := Version(dir)
	require.NoError(t, err)
	require.Equal(t, "9.9.9", got)
}

func TestVersionDisagrees(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cue := "inputs: lucide: {\n\tversion: \"1.57.0\"\n}\n"
	lock := `{
  "dependencies": [
    {
      "kind": "source",
      "ref": "github:lucide-icons/lucide",
      "currentValue": "1.58.0"
    }
  ]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.cue"), []byte(cue), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.lock.json"), []byte(lock), 0o644))
	_, err := Version(dir)
	require.ErrorIs(t, err, errDisagree)
}
