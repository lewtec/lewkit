package terminal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

type base struct{}

func (base) ID() string  { return "terminal" }
func (base) Weight() int { return 10 }

func (base) CheckCompatibility(context.Context) error { return nil }

type chooserFactory struct{ base }

func (chooserFactory) Name() string { return "Terminal (Fuzzy)" }

func (chooserFactory) New(context.Context) (launcher.Chooser, error) {
	return backend{}, nil
}

type prompterFactory struct{ base }

func (prompterFactory) Name() string { return "Terminal (Stdin)" }

func (prompterFactory) New(context.Context) (launcher.Prompter, error) {
	return backend{}, nil
}

type confirmerFactory struct{ base }

func (confirmerFactory) Name() string { return "Terminal (y/n)" }

func (confirmerFactory) New(context.Context) (launcher.Confirmer, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Choose(_ context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	idx, err := fuzzyfinder.Find(
		opts.Items,
		func(i int) string {
			return opts.Items[i].Label
		},
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return nil, nil
		}
		return nil, err
	}
	return &opts.Items[idx], nil
}

func (backend) Prompt(_ context.Context, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", scanner.Err()
}

func (backend) Confirm(_ context.Context, message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		text := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return text == "y" || text == "yes", nil
	}
	return false, scanner.Err()
}
