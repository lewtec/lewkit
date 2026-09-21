// Package profile records runtime pprof data to a directory or an HTTP endpoint.
package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/lewtec/lewkit/x/io"
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

func (p *Profile) file(name string) string {
	return filepath.Join(p.directory, fmt.Sprintf("%s.prof", name))
}

func (p *Profile) Run(ctx context.Context) error {
	if p.address != "" {
		return p.serve(ctx)
	}
	return p.write(ctx)
}

func (p *Profile) write(ctx context.Context) error {
	if err := io.Mkdirp(p.directory); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	cpuFile, err := os.Create(p.file("cpu"))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	defer cpuFile.Close()
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	defer pprof.StopCPUProfile()

	for _, prof := range pprof.Profiles() {
		profileFile, err := os.Create(p.file(prof.Name()))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrProfileWrite, err)
		}
		defer profileFile.Close()
		defer prof.WriteTo(profileFile, 0)
	}
	<-ctx.Done()
	return nil
}

// Handler returns the /debug/pprof/ mux used when Profile is an address.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", httppprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", httppprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", httppprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", httppprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", httppprof.Trace)
	return mux
}

func (p *Profile) serve(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.address)
	if err != nil {
		return fmt.Errorf("pprof listen: %w", err)
	}
	slog.Debug("pprof listening", "addr", listener.Addr().String())
	server := &http.Server{
		Handler: Handler(),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			serveErr <- nil
			return
		}
		serveErr <- err
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			shutdownErr = errors.Join(shutdownErr, server.Close())
		}
		if err := <-serveErr; err != nil {
			return fmt.Errorf("pprof serve: %w", errors.Join(shutdownErr, err))
		}
		if shutdownErr != nil {
			return fmt.Errorf("pprof serve: %w", shutdownErr)
		}
		return nil
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("pprof serve: %w", err)
		}
		return nil
	}
}
