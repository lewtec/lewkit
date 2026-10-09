package cmd

import (
	"errors"
	"fmt"
	"regexp"
)

// errEmptyPattern rejects "" because regexp.Compile("") matches every string.
var errEmptyPattern = errors.New("empty pattern")

// RegexpArg is a RE2 pattern. MatchString is unanchored unless the pattern uses ^ or $.
type RegexpArg struct {
	Container[*regexp.Regexp]
}

func (a *RegexpArg) Parse(arg string) error {
	if arg == "" {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, errEmptyPattern)
	}
	re, err := regexp.Compile(arg)
	return setParsed(&a.Container, re, err)
}

// MatchString reports whether s matches the pattern.
// A zero RegexpArg does not match.
func (a RegexpArg) MatchString(s string) bool {
	re := a.Value()
	if re == nil {
		return false
	}
	return re.MatchString(s)
}

var (
	_ Parser              = (*RegexpArg)(nil)
	_ Arg[*regexp.Regexp] = (*RegexpArg)(nil)
)
