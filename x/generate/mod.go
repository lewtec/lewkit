// Package generate holds helpers shared by generate tools.
package generate

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
)

var (
	// ErrDirRequired is an empty directory argument.
	ErrDirRequired = errors.New("directory required")
	// ErrNoModule is a go.mod with no module line.
	ErrNoModule = errors.New("go.mod: no module line")
	// ErrNoGoMod is a tree with no go.mod ancestor.
	ErrNoGoMod = errors.New("no go.mod")
)

// ModuleLine returns the module path from go.mod contents.
func ModuleLine(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		if line, ok := strings.CutPrefix(scanner.Text(), "module "); ok {
			return strings.TrimSpace(line), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", ErrNoModule
}
