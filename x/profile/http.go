package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"time"
)

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
