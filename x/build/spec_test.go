package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecLoadMergesFlagsOverJSON(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"version_name": "0.1.0",
		"go_main": "."
	}`), 0o644)
	require.NoError(t, err)

	cfg, _, err := Spec{Dir: dir, ID: "br.tec.lew.flag", Version: "1.2.3"}.Load()
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.flag", cfg.PackageID)
	require.Equal(t, "File", cfg.AppName)
	require.Equal(t, "1.2.3", cfg.VersionName)
}
