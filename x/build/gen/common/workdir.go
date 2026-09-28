package common

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveWorkDir returns an existing directory, or a new temp dir when workDir is empty.
// cleanup is true only when a temp dir was created.
func ResolveWorkDir(workDir, tempPattern string) (dir string, cleanup bool, err error) {
	if strings.TrimSpace(workDir) != "" {
		abs, err := filepath.Abs(workDir)
		if err != nil {
			return "", false, err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return "", false, err
		}
		return abs, false, nil
	}
	dir, err = os.MkdirTemp("", tempPattern)
	if err != nil {
		return "", false, err
	}
	return dir, true, nil
}
