package release

import (
	"os"
	"regexp"
	"strings"
)

// name is the short product name.
// A release stamp sets it with -X github.com/lewtec/lewkit/x/release.name=myapp.
// An empty stamp falls back to LEWKIT_NAME, then defaultName.
var name string

// defaultName is the product name when the stamp and LEWKIT_NAME are empty or invalid.
const defaultName = "lewkit"

// namePattern is one identifier: a letter or underscore, then letters, digits, or underscores.
// Paths, script bridges, and protocol names all use that shape.
var namePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// Name is the short product name for paths, protocol names, and titles.
// The name stamp wins, then LEWKIT_NAME. An empty or invalid value keeps [defaultName].
// A caller's own field still overrides this.
func Name() string {
	if n := validName(name); n != "" {
		return n
	}
	if n := validName(os.Getenv("LEWKIT_NAME")); n != "" {
		return n
	}
	return defaultName
}

func validName(s string) string {
	s = strings.TrimSpace(s)
	if namePattern.MatchString(s) {
		return s
	}
	return ""
}
