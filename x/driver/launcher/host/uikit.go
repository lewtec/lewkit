package host

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/hostask"
	"github.com/lewtec/lewkit/x/driver/internal/uikitapp"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

type uikitAsk struct{}

func (uikitAsk) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	raw, err := uikitapp.Ask(ctx, hostask.KindChoose, opts.Prompt, launcher.ChoiceLines(opts.Items))
	if err != nil {
		return nil, err
	}
	return launcher.ChooseAnswer(opts.Items, raw)
}

func (uikitAsk) Prompt(ctx context.Context, prompt string) (string, error) {
	raw, err := uikitapp.Ask(ctx, hostask.KindPrompt, prompt, "")
	if err != nil {
		return "", err
	}
	return launcher.PromptAnswer(raw)
}

func (uikitAsk) Confirm(ctx context.Context, message string) (bool, error) {
	raw, err := uikitapp.Ask(ctx, hostask.KindConfirm, message, "")
	if err != nil {
		return false, err
	}
	return launcher.ConfirmAnswer(raw)
}

func init() {
	launcher.RegisterAsk("launcher_uikit", 80, "UIKit (Choose)", "UIKit (Prompt)", "UIKit (Confirm)", func(context.Context) error {
		return uikitapp.RequireProcess()
	}, func(context.Context) (uikitAsk, error) {
		return uikitAsk{}, nil
	})
}
