package toolexec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlattenReadsAResponseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "resp")
	require.NoError(t, os.WriteFile(path, []byte("\"-p\"\n\nexample.com/pkg\n"), 0o644))
	got := Flatten([]string{"-o", "a.o", "@" + path, "@"})
	require.Equal(t, []string{"-o", "a.o", "-p", "example.com/pkg", "@"}, got)
	require.Equal(t, []string{"@missing"}, Flatten([]string{"@missing"}))
}

func TestToolNameStripsExe(t *testing.T) {
	require.Equal(t, "compile", ToolName("/usr/lib/compile.exe"))
	require.Equal(t, "compile", ToolName("/usr/lib/compile"))
}
