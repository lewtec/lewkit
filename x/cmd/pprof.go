package cmd

import (
	"context"
	"log/slog"

	"github.com/lewtec/lewkit/x/profile"
)

// PprofArg is a pprof directory or listen address. Empty disables profiling.
type PprofArg struct {
	Container[string]
	profile profile.Profile
}

func (p *PprofArg) Parse(arg string) error {
	p.value = arg
	if arg == "" {
		p.profile = profile.Profile{}
		return nil
	}
	var listen AddrArg
	if err := listen.Parse(arg); err == nil {
		p.profile = profile.Address(listen.Value())
		return nil
	}
	p.profile = profile.Directory(arg)
	return nil
}

func (p PprofArg) Directory() string { return p.profile.Directory() }

func (p PprofArg) Address() string { return p.profile.Address() }

// Start runs pprof in a goroutine until ctx is done. No-op when unset.
func (p *PprofArg) Start(ctx context.Context) {
	if p.value == "" {
		return
	}
	go func() {
		if err := p.profile.Run(ctx); err != nil {
			slog.Error(err.Error())
		}
	}()
}

var (
	_ Parser      = (*PprofArg)(nil)
	_ Arg[string] = (*PprofArg)(nil)
)
