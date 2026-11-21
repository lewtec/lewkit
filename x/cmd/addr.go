package cmd

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

var (
	errEmptyAddress = errors.New("empty address")
	errMissingPort  = errors.New("missing port")
)

// AddrArg is a host:port listen address. A bare port (as in $PORT) is :port.
type AddrArg struct {
	Container[string]
}

func (a *AddrArg) Parse(arg string) error {
	if arg == "" {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, errEmptyAddress)
	}
	host, port, err := net.SplitHostPort(arg)
	if err != nil {
		if _, perr := strconv.ParseUint(arg, 10, 16); perr != nil {
			return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
		}
		host, port = "", arg
	} else if port == "" {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, errMissingPort)
	}
	a.value = net.JoinHostPort(host, port)
	return nil
}

var (
	_ Parser      = (*AddrArg)(nil)
	_ Arg[string] = (*AddrArg)(nil)
)
