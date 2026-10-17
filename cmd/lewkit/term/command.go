// Package term is the lewkit term command.
package term

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	mux "github.com/lewtec/lewkit/x/term"
)

// Cmd is lewkit term.
type Cmd struct {
	serve  *serveCmd
	attach *attachCmd
	rpc    *rpcCmd
}

// Description is the command summary.
func (Cmd) Description() string {
	return "multiplex terminals"
}

type serveCmd struct {
	socket cmd.StringArg `long:"socket" default:"" help:"unix socket path"`
}

func (serveCmd) Description() string {
	return "serve the ubersession"
}

func (c *serveCmd) Run(ctx context.Context) error {
	path, err := socketPath(c.socket)
	if err != nil {
		return err
	}
	return mux.Listen(ctx, path)
}

type attachCmd struct {
	socket cmd.StringArg `long:"socket" default:"" help:"unix socket path"`
	pane   cmd.StringArg `long:"pane" default:"" help:"follow this pane instead of the ubersession"`
}

func (attachCmd) Description() string {
	return "attach with the command palette"
}

func (c *attachCmd) Run(ctx context.Context) error {
	return entry.RunAfterView(ctx, func(ctx context.Context) error {
		path, err := socketPath(c.socket)
		if err != nil {
			return err
		}
		client, err := mux.Dial(ctx, path)
		if err != nil {
			return err
		}
		defer client.Close()
		return mux.Run(ctx, client, c.pane.Value())
	})
}

type rpcCmd struct {
	socket cmd.StringArg  `long:"socket" default:"" help:"unix socket path"`
	method cmd.StringArg  `help:"method, such as pane.list"`
	params *cmd.StringArg `help:"JSON params object"`
}

func (rpcCmd) Description() string {
	return "call one ubersession method"
}

func (c *rpcCmd) Run(ctx context.Context) error {
	method, err := mux.ParseMethod(c.method.Value())
	if err != nil {
		return err
	}
	var params any
	if c.params != nil && c.params.Value() != "" {
		if !json.Valid([]byte(c.params.Value())) {
			return fmt.Errorf("%w: %s", mux.ErrProtocol, "params")
		}
		params = json.RawMessage(c.params.Value())
	}
	path, err := socketPath(c.socket)
	if err != nil {
		return err
	}
	client, err := mux.Dial(ctx, path)
	if err != nil {
		return err
	}
	defer client.Close()
	var result json.RawMessage
	if err := client.Call(ctx, method, params, &result); err != nil {
		return err
	}
	if len(result) == 0 {
		result = json.RawMessage("{}")
	}
	_, err = fmt.Printf("%s\n", result)
	return err
}

func socketPath(arg cmd.StringArg) (string, error) {
	if arg.Value() != "" {
		return arg.Value(), nil
	}
	return mux.Socket()
}
