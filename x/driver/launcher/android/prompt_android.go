//go:build android && cgo

package android

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/entry"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[launcher.Prompter](factory{}) }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (launcher.Prompter, error) { return backend{}, nil }

type backend struct{}

var promptGate sync.Mutex

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	on, err := androidffi.OnLooper()
	if err != nil {
		return "", err
	}
	if on {
		return "", fmt.Errorf("%w: main looper", driver.ErrIncompatible)
	}

	promptGate.Lock()
	defer promptGate.Unlock()

	ch := make(chan promptReply, 1)
	entry.HandlePrompt(func(text string, code int) {
		select {
		case ch <- promptReply{text: text, code: code}:
		default:
		}
	})
	defer entry.HandlePrompt(nil)

	if err := androidffi.StaticVoid("prompt", "(Ljava/lang/String;)V", prompt); err != nil {
		return "", err
	}
	text, err := takePrompt(ctx, ch)
	if err != nil && ctx.Err() != nil {
		if dismissErr := androidffi.StaticVoid("dismissPrompt", "()V"); dismissErr != nil {
			return "", errors.Join(err, dismissErr)
		}
	}
	return text, err
}
