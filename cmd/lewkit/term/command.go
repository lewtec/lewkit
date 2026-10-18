// Package term is the lewkit term command.
package term

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	mux "github.com/lewtec/lewkit/x/term"
	"github.com/lewtec/lewkit/x/term/schema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	params *cmd.StringArg `help:"protojson object for the method"`
}

func (rpcCmd) Description() string {
	return "call one ubersession method"
}

func (c *rpcCmd) Run(ctx context.Context) error {
	method, err := mux.ParseMethod(c.method.Value())
	if err != nil {
		return err
	}
	params, err := rpcParams(method, c.params)
	if err != nil {
		return err
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
	resp, err := client.Call(ctx, method, params)
	if err != nil {
		return err
	}
	text, err := payloadJSON(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Printf("%s\n", text)
	return err
}

func rpcParams(method mux.Method, arg *cmd.StringArg) (proto.Message, error) {
	text := ""
	if arg != nil {
		text = arg.Value()
	}
	msg := requestMessage(method)
	if text == "" {
		return msg, nil
	}
	if msg == nil {
		return nil, fmt.Errorf("%w: %s", mux.ErrProtocol, "params")
	}
	err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(text), msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", mux.ErrProtocol, err)
	}
	return msg, nil
}

func requestMessage(method mux.Method) proto.Message {
	switch method {
	case mux.MethodPaneGet, mux.MethodPaneKill:
		return &schema.PaneId{}
	case mux.MethodPaneCreate:
		return &schema.CreateRequest{}
	case mux.MethodPaneWrite:
		return &schema.WriteRequest{}
	case mux.MethodWatch:
		return &schema.WatchRequest{}
	default:
		return nil
	}
}

func payloadJSON(resp *schema.Envelope) (string, error) {
	payload := mux.Payload(resp)
	if payload == nil {
		return "{}", nil
	}
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("%w: %w", mux.ErrProtocol, err)
	}
	return string(body), nil
}

func socketPath(arg cmd.StringArg) (string, error) {
	if arg.Value() != "" {
		return arg.Value(), nil
	}
	return mux.Socket()
}
