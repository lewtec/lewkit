package view

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Serve serves handler on listener until ctx is done.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveError := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveError <- err
	}()
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return fmt.Errorf("stop preview: %w", err)
		}
		return <-serveError
	case err := <-serveError:
		if err != nil {
			return fmt.Errorf("preview: %w", err)
		}
		return nil
	}
}
