//go:build android && cgo

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/androidask"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func open(context.Context) (backend, error) { return backend{}, nil }

type backend struct{}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	raw, err := androidask.Call(ctx, "choose", opts.Prompt, launcher.ChoiceLines(opts.Items))
	if err != nil {
		return nil, err
	}
	return launcher.ChooseAnswer(opts.Items, raw)
}

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	raw, err := androidask.Call(ctx, "prompt", prompt, "", "")
	if err != nil {
		return "", err
	}
	return launcher.PromptAnswer(raw)
}

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	raw, err := androidask.Call(ctx, "confirm", message, "")
	if err != nil {
		return false, err
	}
	return launcher.ConfirmAnswer(raw)
}
