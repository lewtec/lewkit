// Command decompview serves a read-only browser preview of a sqlite program database.
//
// Cmd+K and Ctrl+K open a jump palette for artifacts and functions.
// The page shows the summary, the listing, the comment, and matched source.
// The file is opened with decomp.Open. The preview does not disassemble or match.
//
//	decompview PROJECT
//	decompview --listen 127.0.0.1:8766 PROJECT
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/decomp"
	"github.com/lewtec/lewkit/x/entry"
)

func main() {
	application, err := cmd.Parse[cmd.App[arguments]](os.Args[1:]...)
	if err != nil {
		entry.Main(context.Background(), func(context.Context) error { return err })
		return
	}
	entry.Main(context.Background(), application.Run)
}

type arguments struct {
	listen  cmd.StringArg `long:"listen" default:"127.0.0.1:8766" help:"preview address"`
	project cmd.StringArg `help:"sqlite program database"`
}

func (arguments) Description() string {
	return "preview a decomp project in the browser"
}

func (command *arguments) Run(ctx context.Context) (err error) {
	path := command.project.Value()
	if path == "" {
		return cmd.ErrUsage
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("project: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("project %s is a directory", path)
	}
	project, err := decomp.Open(ctx, path)
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()

	listener, err := net.Listen("tcp", command.listen.Value())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	fmt.Fprintf(os.Stdout, "http://%s/\n", listener.Addr().String())
	return serveListener(ctx, listener, newPreview(project))
}

func serveListener(ctx context.Context, listener net.Listener, handler http.Handler) error {
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

func joinClose(err, closeErr error) error {
	if err != nil {
		return err
	}
	return closeErr
}
