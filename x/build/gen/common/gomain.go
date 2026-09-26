package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	goMain = strings.TrimSpace(goMain)
	if goMain == "" {
		goMain = "."
	}
	if !filepath.IsAbs(goMain) {
		goMain = filepath.Join(baseDir, goMain)
	}
	abs, err := filepath.Abs(goMain)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("go_main %q: %w", abs, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrGoMainNotDir, abs)
	}
	return abs, nil
}
