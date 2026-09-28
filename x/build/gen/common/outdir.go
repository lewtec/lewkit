package common

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// HostConfigFile is eletrocromo.json in a generated host tree.
const HostConfigFile = "eletrocromo.json"

// ResolveCreateOut trims outDir, absolutizes it, and PrepareOutDir.
func ResolveCreateOut(outDir string, force bool) (string, error) {
	out := strings.TrimSpace(outDir)
	if out == "" {
		return "", ErrOutDirRequired
	}
	out, err := filepath.Abs(out)
	if err != nil {
		return "", err
	}
	if err := PrepareOutDir(out, force); err != nil {
		return "", err
	}
	return out, nil
}

// WriteHostJSON writes eletrocromo.json under out.
func WriteHostJSON(out string, raw []byte) error {
	return os.WriteFile(filepath.Join(out, HostConfigFile), raw, 0o644)
}

// PrepareOutDir creates out or, with force, wipes an existing directory's contents.
func PrepareOutDir(out string, force bool) error {
	st, err := os.Stat(out)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return os.MkdirAll(out, 0o755)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("%w: %s", ErrOutPathNotDir, out)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("%w: %s", ErrOutDirNotEmpty, out)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(out, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
