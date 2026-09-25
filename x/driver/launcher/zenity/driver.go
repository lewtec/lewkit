package zenity

import (
	"context"
	"os/exec"
	"strings"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

type base struct{}

func (base) ID() string  { return "zenity" }
func (base) Weight() int { return 40 }

func (base) CheckCompatibility(ctx context.Context) error {
	return launcher.RequireDisplayBinary(ctx, "zenity")
}

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "Zenity (Prompt)" }

func (prompterFactory) New(context.Context) (launcher.Prompter, error) {
	return backend{}, nil
}

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "Zenity (Confirm)" }

func (confirmerFactory) New(context.Context) (launcher.Confirmer, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	out, err := exec.CommandContext(ctx, "zenity", "--entry", "--text", prompt).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	err := exec.CommandContext(ctx, "zenity", "--question", "--text", message).Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}
