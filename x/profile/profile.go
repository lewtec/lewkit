// Package profile records runtime pprof data to a directory or an HTTP endpoint.
package profile

import (
	"context"
	"errors"
)

var (
	ErrProfileWrite = errors.New("can't write profile")
)

// Directory writes cpu.prof and a snapshot of each named profile when ctx ends.
func Directory(path string) Profile {
	return Profile{directory: path}
}

// Address serves /debug/pprof/ on addr until ctx ends.
func Address(addr string) Profile {
	return Profile{address: addr}
}

type Profile struct {
	directory string
	address   string
}

func (p Profile) Directory() string { return p.directory }

func (p Profile) Address() string { return p.address }

func (p *Profile) Run(ctx context.Context) error {
	if p.address != "" {
		return p.serve(ctx)
	}
	if p.directory == "" {
		return nil
	}
	return p.write(ctx)
}
