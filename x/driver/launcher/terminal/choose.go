//go:build !android

package terminal

import (
	"context"
	"errors"

	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/lewtec/lewkit/x/driver/launcher"
)

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
