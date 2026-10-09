//go:build android

package androidask

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

var errMainLooper = errors.New("main looper")

// Platform reports whether a Java VM can present lewkit.Ask.
func Platform(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

// Call sets the listener, invokes method, and waits for one askwire reply.
// args are the Java arguments after the method name.
func Call(ctx context.Context, method string, args ...any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if on, err := androidffi.OnLooper(); err == nil && on {
		return "", fmt.Errorf("%w: %w", driver.ErrUnavailable, errMainLooper)
	}
	ch, token, err := begin()
	if err != nil {
		return "", fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	defer end()

	proxy, err := jni.Proxy("java.util.function.Consumer", func(name string, got []any) (any, error) {
		if name != "accept" {
			return nil, nil
		}
		text := ""
		if len(got) == 1 {
			if s, ok := got[0].(string); ok {
				text = s
			}
		}
		deliver(token, text)
		return nil, nil
	})
	if err != nil {
		return "", err
	}
	defer proxy.Release()
	if _, err = jni.CallStatic("lewkit.Ask", "setListener", proxy); err != nil {
		return "", err
	}
	defer func() {
		_, _ = jni.CallStatic("lewkit.Ask", "setListener", nil)
	}()
	callArgs := append([]any{}, args...)
	if _, err = jni.CallStatic("lewkit.Ask", method, callArgs...); err != nil {
		return "", err
	}
	select {
	case got := <-ch:
		return got, nil
	case <-ctx.Done():
		_, _ = jni.CallStatic("lewkit.Ask", "cancel")
		return "", ctx.Err()
	}
}
