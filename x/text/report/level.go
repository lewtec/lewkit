package report

import (
	"fmt"
	"strconv"
	"strings"
)

// Level is a diagnostic severity. The wire form matches SARIF result.level.
type Level string

const (
	LevelError   Level = "error"
	LevelWarning Level = "warning"
	LevelNote    Level = "note"
)

// String returns error, warning, or note.
func (l Level) String() string { return string(l) }

// ParseLevel normalizes s. Empty means LevelWarning.
// Accepted values are error, warning, and note, in any case.
func ParseLevel(s string) (Level, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return LevelWarning, nil
	}
	l := Level(s)
	switch l {
	case LevelError, LevelWarning, LevelNote:
		return l, nil
	default:
		return "", fmt.Errorf("%w %s (want error, warning, or note)", ErrLevel, strconv.Quote(s))
	}
}
