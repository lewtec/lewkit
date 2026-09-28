package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

const (
	promptCancel     = 0
	promptOK         = 1
	promptNoActivity = 2
)

type promptReply struct {
	text string
	code int
}

func decodePrompt(text string, code int) (string, error) {
	switch code {
	case promptOK:
		return text, nil
	case promptCancel:
		return "", launcher.ErrCanceled
	case promptNoActivity:
		return "", fmt.Errorf("%w: no foreground activity", driver.ErrUnavailable)
	default:
		return "", fmt.Errorf("%w: prompt code %d", driver.ErrUnavailable, code)
	}
}

func takePrompt(ctx context.Context, ch <-chan promptReply) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return decodePrompt(res.text, res.code)
	}
}
