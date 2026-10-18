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
	errBadPort      = errors.New("invalid port")
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
	} else if _, perr := strconv.ParseUint(port, 10, 16); perr != nil {
		// A Windows path contains a colon. Its tail is not a port.
		return fmt.Errorf("%w: %w", ErrInvalidArgument, errMissingPort)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, errBadPort)
	}
	a.value = net.JoinHostPort(host, port)
	return nil
}

var (
	_ Parser      = (*AddrArg)(nil)
	_ Arg[string] = (*AddrArg)(nil)
)
