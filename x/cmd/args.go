package cmd

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"golang.org/x/exp/constraints"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUnknownFlag     = errors.New("unknown flag")
	ErrMissingValue    = errors.New("missing value")
	ErrInvalidSpec     = errors.New("invalid command spec")
	ErrUnknownCommand  = errors.New("unknown command")
	errEmptyAddress    = errors.New("empty address")
	errMissingPort     = errors.New("missing port")
)

type Parser interface {
	Parse(string) error
}

type Counter interface {
	Count(int) error
}

type Valuer[T any] interface {
	Value() T
}

// ArgDefaulter supplies a default when the field has no default tag.
type ArgDefaulter interface {
	ArgDefault() string
}

func Values[T any, A Valuer[T]](in []A) []T {
	out := make([]T, len(in))
	for i, a := range in {
		out[i] = a.Value()
	}
	return out
}

type Arg[T any] interface {
	Parser
	Valuer[T]
}

// KV is a key/value product. V is an arg, e.g. *StringArg or *IntArg[int].
type KV[T any, V Arg[T]] struct {
	K StringArg
	V V
}

// Map extracts keys and Value()s from a slice (or Seq) of KV.
func Map[T any, V Arg[T], S ~[]KV[T, V]](in S) map[string]T {
	out := make(map[string]T, len(in))
	for _, kv := range in {
		out[kv.K.Value()] = kv.V.Value()
	}
	return out
}

type Container[T any] struct {
	value T
}

func (c Container[T]) Value() T {
	return c.value
}

type StringArg struct {
	Container[string]
}

func (s *StringArg) Parse(arg string) error {
	s.value = arg
	return nil
}

type IntArg[T constraints.Integer] struct {
	Container[T]
}

func (i *IntArg[T]) Parse(arg string) error {
	value, err := strconv.ParseInt(arg, 0, 64)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	i.value = T(value)
	return err
}

type FloatArg[T constraints.Float] struct {
	Container[T]
}

func (i *FloatArg[T]) Parse(arg string) error {
	value, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	i.value = T(value)
	return err
}

type Flag struct {
	Container[bool]
}

func (Flag) ArgDefault() string { return "false" }

func (f *Flag) Count(times int) error {
	f.value = times != 0
	if times > 1 {
		return fmt.Errorf("%w: flags must only be used once", ErrInvalidArgument)
	}
	return nil
}

type Count struct {
	Container[int]
}

func (Count) ArgDefault() string { return "0" }

func (f *Count) Count(times int) error {
	f.value = times
	return nil
}

func (f *Count) Parse(arg string) error {
	value, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	if value < 0 {
		return fmt.Errorf("%w: count must be non-negative", ErrInvalidArgument)
	}
	f.value = value
	return nil
}

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
	_ Parser       = (*StringArg)(nil)
	_ Parser       = (*IntArg[int])(nil)
	_ Parser       = (*FloatArg[float64])(nil)
	_ Parser       = (*Count)(nil)
	_ Parser       = (*AddrArg)(nil)
	_ Counter      = (*Flag)(nil)
	_ Counter      = (*Count)(nil)
	_ Valuer[bool] = (*Flag)(nil)
	_ ArgDefaulter = Flag{}
	_ ArgDefaulter = Count{}
	_ Arg[string]  = (*StringArg)(nil)
	_ Arg[int]     = (*IntArg[int])(nil)
	_ Arg[int]     = (*Count)(nil)
	_ Arg[string]  = (*AddrArg)(nil)
)

// Seq is a greedy positional list of T (zero or more). An untagged []T field
// is the same combinator.
type Seq[T any] []T

// Dash consumes a "--" token and splits the consumers around it.
type Dash struct{}
