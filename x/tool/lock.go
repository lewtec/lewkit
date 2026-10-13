package tool

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// LockSpec is the tool spec pinned in a modot.lock.json at directory or a parent.
// A missing pin is ref@latest. A directory that cannot be opened is skipped.
// A lock that is not JSON is reported as modot.lock.json.
func LockSpec(directory, ref string) (string, error) {
	for {
		spec, found, err := lockSpecIn(directory, ref)
		if err != nil {
			return "", err
		}
		if found {
			return spec, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return ref + "@latest", nil
		}
		directory = parent
	}
}

func lockSpecIn(directory, ref string) (string, bool, error) {
	root, err := lewpath.Open(directory)
	if err != nil {
		return "", false, nil
	}
	defer root.Close()
	name := lewpath.New("modot.lock.json")
	isFile, err := name.IsFile(root)
	if err != nil || !isFile {
		return "", false, err
	}
	body, err := name.ReadFile(root)
	if err != nil {
		return "", false, err
	}
	var lock struct {
		Dependencies []struct {
			Kind         string `json:"kind"`
			Ref          string `json:"ref"`
			CurrentValue string `json:"currentValue"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return "", false, fmt.Errorf("modot.lock.json: %w", err)
	}
	for _, dependency := range lock.Dependencies {
		if dependency.Kind == "tool" && dependency.Ref == ref && dependency.CurrentValue != "" {
			return ref + "@" + dependency.CurrentValue, true, nil
		}
	}
	return "", false, nil
}
