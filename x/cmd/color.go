package cmd

import (
	"fmt"
	"strings"

	lewimage "github.com/lewtec/lewkit/x/image"
)

// RGBArg is a [lewimage.RGB] flag. Values are #RGB, #RRGGBB, or #RRGGBBAA.
// An empty value means the flag was omitted.
type RGBArg struct {
	Container[lewimage.RGB]
	set bool
}

func (c *RGBArg) Parse(arg string) error {
	if strings.TrimSpace(arg) == "" {
		return nil
	}
	value, err := lewimage.ParseRGB(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	c.value = value
	c.set = true
	return nil
}

// IsSet reports whether Parse saw a color.
func (c RGBArg) IsSet() bool { return c.set }

var (
	_ Parser            = (*RGBArg)(nil)
	_ Arg[lewimage.RGB] = (*RGBArg)(nil)
)
