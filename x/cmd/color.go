package cmd

import (
	"fmt"
	"strings"

	lewimage "github.com/lewtec/lewkit/x/image"
)

// ColorArg is a [lewimage.Color] flag. Values are #RGB, #RRGGBB, or #RRGGBBAA.
// An empty value means the flag was omitted.
type ColorArg struct {
	Container[lewimage.Color]
	set bool
}

func (c *ColorArg) Parse(arg string) error {
	if strings.TrimSpace(arg) == "" {
		return nil
	}
	value, err := lewimage.ParseColor(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	c.value = value
	c.set = true
	return nil
}

// IsSet reports whether Parse saw a color.
func (c ColorArg) IsSet() bool { return c.set }

var (
	_ Parser              = (*ColorArg)(nil)
	_ Arg[lewimage.Color] = (*ColorArg)(nil)
)
