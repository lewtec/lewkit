// Package host asks the packaged iOS or macOS host to present a dialog.
// Android uses lewkit.Ask instead. The host watches the ask directory.
package host

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/hostask"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

type base struct{}

func (base) ID() string  { return "launcher_host" }
func (base) Weight() int { return 70 }

func (base) CheckCompatibility(context.Context) error {
	if runtime.GOOS == "android" {
		return fmt.Errorf("%w: android", driver.ErrIncompatible)
	}
	if !hostask.Available() {
		return fmt.Errorf("%w: no ask host", driver.ErrIncompatible)
	}
	return nil
}

type chooserFactory struct{ base }

func (chooserFactory) Name() string { return "Host (Choose)" }

func (chooserFactory) New(context.Context) (launcher.Chooser, error) { return backend{}, nil }

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "Host (Prompt)" }

func (prompterFactory) New(context.Context) (launcher.Prompter, error) { return backend{}, nil }

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "Host (Confirm)" }

func (confirmerFactory) New(context.Context) (launcher.Confirmer, error) { return backend{}, nil }

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
	driver.Register[launcher.Chooser](chooserFactory{})
	driver.Register[launcher.Prompter](prompterFactory{})
	driver.Register[launcher.Confirmer](confirmerFactory{})
}

var (
	_ driver.DriverFactory[launcher.Chooser]   = chooserFactory{}
	_ driver.DriverFactory[launcher.Prompter]  = prompterFactory{}
	_ driver.DriverFactory[launcher.Confirmer] = confirmerFactory{}
	_ driver.Weighter                          = base{}
)
