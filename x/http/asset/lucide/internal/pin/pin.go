// Package pin reads the Lucide release from the modot input.
// modot.cue names the tag. modot.lock.json records that same tag.
package pin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
)

const (
	inputName = "lucide"
	inputRef  = "github:lucide-icons/lucide"
)

var (
	errModule   = errs.Wrap(errs.ErrMissing, "module root")
	errCue      = errs.Wrap(errs.ErrMissing, "modot.cue lucide input")
	errLock     = errs.Wrap(errs.ErrMissing, "modot.lock.json lucide input")
	errLockJSON = errs.Wrap(errs.ErrSyntax, "modot.lock.json")
	errDisagree = errs.Wrap(errs.ErrInvalid, "lucide version")
)

// Version is the Lucide tag when modot.cue and modot.lock.json name the same one.
func Version(moduleRoot string) (string, error) {
	cueText, err := os.ReadFile(filepath.Join(moduleRoot, "modot.cue"))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errCue, err)
	}
	want, err := cueInputVersion(string(cueText), inputName)
	if err != nil {
		return "", err
	}
	lockText, err := os.ReadFile(filepath.Join(moduleRoot, "modot.lock.json"))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errLock, err)
	}
	got, err := lockInputVersion(lockText, inputRef)
	if err != nil {
		return "", err
	}
	if want != got {
		return "", fmt.Errorf("%w: modot.cue %s, lock %s", errDisagree, want, got)
	}
	return want, nil
}

// ModuleRoot walks up from start to the module that contains modot.cue.
// An empty start uses the working directory.
func ModuleRoot(start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	dir := start
	for {
		_, modErr := os.Stat(filepath.Join(dir, "go.mod"))
		_, cueErr := os.Stat(filepath.Join(dir, "modot.cue"))
		if modErr == nil && cueErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errModule
		}
		dir = parent
	}
}

func cueInputVersion(text, name string) (string, error) {
	header := regexp.MustCompile(`(?m)^inputs:\s*` + regexp.QuoteMeta(name) + `\s*:\s*\{`)
	loc := header.FindStringIndex(text)
	if loc == nil {
		return "", errCue
	}
	rest := text[loc[1]:]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		return "", errCue
	}
	versionLine := regexp.MustCompile(`(?m)^\s*version:\s*"([^"]+)"\s*$`)
	match := versionLine.FindStringSubmatch(rest[:end])
	if match == nil {
		return "", errCue
	}
	return match[1], nil
}

func lockInputVersion(body []byte, ref string) (string, error) {
	var lock struct {
		Dependencies []struct {
			Kind         string `json:"kind"`
			Ref          string `json:"ref"`
			CurrentValue string `json:"currentValue"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return "", fmt.Errorf("%w: %w", errLockJSON, err)
	}
	for _, dep := range lock.Dependencies {
		if dep.Kind == "source" && dep.Ref == ref && dep.CurrentValue != "" {
			return dep.CurrentValue, nil
		}
	}
	return "", errLock
}
