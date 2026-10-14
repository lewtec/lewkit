// Package host asks the packaged iOS or macOS host to present a dialog.
// Android uses lewkit.Ask instead. The host watches the ask directory.
package host

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/hostask"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

type backend struct{}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	raw, err := hostask.Call(ctx, hostask.KindChoose, opts.Prompt, launcher.ChoiceLines(opts.Items))
	if err != nil {
		return nil, err
	}
	return launcher.ChooseAnswer(opts.Items, raw)
}

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	raw, err := hostask.Call(ctx, hostask.KindPrompt, prompt, "")
	if err != nil {
		return "", err
	}
	return launcher.PromptAnswer(raw)
}

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	raw, err := hostask.Call(ctx, hostask.KindConfirm, message, "")
	if err != nil {
		return false, err
	}
	return launcher.ConfirmAnswer(raw)
}

func init() {
	launcher.RegisterAsk("launcher_host", 70, "Host (Choose)", "Host (Prompt)", "Host (Confirm)", func(context.Context) error {
		return hostask.Require()
	}, func(context.Context) (backend, error) {
		return backend{}, nil
	})
}
