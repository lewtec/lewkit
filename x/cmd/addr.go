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
	host string
	port string
}

func (a *AddrArg) Parse(arg string) error {
	host, port, err := parseAddr(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	a.host = host
	a.port = port
	return nil
}

func (a AddrArg) Value() string { return net.JoinHostPort(a.host, a.port) }

func (a AddrArg) Host() string { return a.host }

func (a AddrArg) Port() string { return a.port }

func parseAddr(s string) (string, string, error) {
	if s == "" {
		return "", "", errEmptyAddress
	}
	host, port, err := net.SplitHostPort(s)
	if err == nil {
		if port == "" {
			return "", "", errMissingPort
		}
		return host, port, nil
	}
	if _, perr := strconv.ParseUint(s, 10, 16); perr == nil {
		return "", s, nil
	}
	return "", "", err
}

var (
	_ Parser      = (*AddrArg)(nil)
	_ Arg[string] = (*AddrArg)(nil)
)
