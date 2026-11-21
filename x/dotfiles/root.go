// Package dotfiles locates the user's dotfiles checkout.
//
// [Root] walks [Candidates] and returns the first directory that exists.
// The order matches workspaced: a Codespaces persisted share, then
// ~/.dotfiles, then /etc/.dotfiles.
package dotfiles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned when no candidate directory exists.
var ErrNotFound = errors.New("dotfiles root not found")

// Candidates is the search order. The first existing directory wins.
// A leading ~ expands with the home passed to [Root]. Other paths expand $VAR.
var Candidates = []string{
	"/workspaces/.codespaces/.persistedshare/dotfiles",
	"~/.dotfiles",
	"/etc/.dotfiles",
}

// Root returns the first existing candidate. home expands a leading ~.
func Root(home string) (string, error) {
	for _, path := range Candidates {
		expanded := expand(path, home)
		info, err := os.Stat(expanded)
		if err == nil && info.IsDir() {
			return expanded, nil
		}
	}
	return "", ErrNotFound
}

// expand matches workspaced ExpandPathIn: ~ and ~/ use home, otherwise $VAR.
func expand(path, home string) string {
	if home != "" && strings.HasPrefix(path, "~") {
		if path == "~" {
			return home
		}
		if path[1] == '/' || path[1] == filepath.Separator {
			return filepath.Join(home, path[2:])
		}
	}
	return os.ExpandEnv(path)
}
