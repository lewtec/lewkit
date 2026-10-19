//go:build !ios

package uikitapp

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

func unavailable(name string) error {
	return fmt.Errorf("%w: %s", driver.ErrUnavailable, name)
}

// Ask presents an alert, confirm, prompt, or choose.
// kind is 1 alert, 2 confirm, 3 prompt, 4 choose.
func Ask(ctx context.Context, _ int, _, _ string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("ask")
}

// PasteText writes the pasteboard.
func PasteText(ctx context.Context, _ string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("pasteboard")
}

// PasteImage writes a PNG to the pasteboard.
func PasteImage(ctx context.Context, _ []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("pasteboard")
}

// Battery reads UIDevice.
func Battery(ctx context.Context) (string, int, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	return "", 0, unavailable("battery")
}

// Appearance reads the interface style.
func Appearance(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("appearance")
}

// Open opens a URL or file.
func Open(ctx context.Context, _ string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("open")
}

// Notify posts a local notification.
func Notify(ctx context.Context, _, _, _ string, _ uint32) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("notification")
}

// Pick shows the document picker.
func Pick(ctx context.Context, _ PickRequest) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", unavailable("picker")
}

// Share presents the share sheet.
func Share(ctx context.Context, _, _, _ string, _ []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return unavailable("share")
}
