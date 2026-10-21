//go:build ignore

// genversion writes version_gen.go from the Lucide tag in modot.lock.json.
package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	errModuleRoot = errs.Wrap(errs.ErrMissing, "go.mod")
	errLockJSON   = errs.Wrap(errs.ErrInvalid, "modot.lock.json")
	errLucideTag  = errs.Wrap(errs.ErrMissing, "github:lucide-icons/lucide tag")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "genversion: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	tag, err := lucideTag(filepath.Join(root, "modot.lock.json"))
	if err != nil {
		return err
	}
	raw := fmt.Sprintf(`// Code generated from modot.lock.json. DO NOT EDIT.

package lucide

// Version is the Lucide release these bodies were taken from.
// The tag is the currentValue of github:lucide-icons/lucide in modot.lock.json.
const Version = %s
`, strconv.Quote(tag))
	body, err := format.Source([]byte(raw))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "x/http/asset/lucide/version_gen.go"), body, 0o644)
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		info, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil && !info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errModuleRoot
		}
		dir = parent
	}
}

func lucideTag(path string) (string, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var lock struct {
		Dependencies []struct {
			Kind         string `json:"kind"`
			Ref          string `json:"ref"`
			CurrentValue string `json:"currentValue"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(text, &lock); err != nil {
		return "", fmt.Errorf("%w: %w", errLockJSON, err)
	}
	for _, dep := range lock.Dependencies {
		if dep.Kind == "source" && dep.Ref == "github:lucide-icons/lucide" && dep.CurrentValue != "" {
			return dep.CurrentValue, nil
		}
	}
	return "", errLucideTag
}
