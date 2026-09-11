package cmd

import (
	"fmt"
	"time"
)

// DurationArg is a time.Duration. Values use Go duration syntax (300ms, 1.5h, 2h45m).
type DurationArg struct {
	Container[time.Duration]
}

func (d *DurationArg) Parse(arg string) error {
	value, err := time.ParseDuration(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	d.value = value
	return nil
}

var (
	_ Parser             = (*DurationArg)(nil)
	_ Arg[time.Duration] = (*DurationArg)(nil)
)
