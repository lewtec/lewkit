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
