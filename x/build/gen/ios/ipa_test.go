package ios

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackWritesPayload(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Demo.app")
	require.NoError(t, os.MkdirAll(app, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(app, "Demo"), []byte("macho"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(app, "Info.plist"), []byte("<plist/>"), 0o644))
	ipa := filepath.Join(t.TempDir(), "nested", "Demo_ios_arm64.ipa")
	require.NoError(t, Pack(t.Context(), app, ipa))

	reader, err := zip.OpenReader(ipa)
	require.NoError(t, err)
	defer reader.Close()
	modes := map[string]os.FileMode{}
	for _, file := range reader.File {
		require.NotContains(t, file.Name, "\\")
		require.True(t, strings.HasPrefix(file.Name, "Payload/"), file.Name)
		modes[file.Name] = file.Mode()
	}
	require.Contains(t, modes, "Payload/")
	require.Contains(t, modes, "Payload/Demo.app/")
	require.Contains(t, modes, "Payload/Demo.app/Info.plist")
	require.Equal(t, os.FileMode(0o755), modes["Payload/Demo.app/Demo"].Perm())
	require.Zero(t, modes["Payload/Demo.app/Info.plist"]&0o111)
	member, err := reader.Open("Payload/Demo.app/Demo")
	require.NoError(t, err)
	defer member.Close()
	body, err := io.ReadAll(member)
	require.NoError(t, err)
	require.Equal(t, "macho", string(body))
}

func TestPackRejectsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "Demo")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	err := Pack(t.Context(), file, filepath.Join(t.TempDir(), "Demo.ipa"))
	require.ErrorIs(t, err, ErrIPA)
}
