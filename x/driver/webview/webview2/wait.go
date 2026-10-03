package webview2

import (
	"context"
	"errors"
)

var errCompletion = errors.New("webview2 completion failed")

// waitFor waits until a WebView2 completion handler signals.
//
// The handler runs on this thread, from a message the runtime posts after
// CreateEnvironment or CreateCoreWebView2Controller returns. pump must
// dispatch that queue. done and failed are buffered so the handler can
// signal without waiting for this goroutine.
func waitFor(ctx context.Context, done <-chan uintptr, failed <-chan error, pump func(context.Context) error) (uintptr, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case value := <-done:
			return value, nil
		case err := <-failed:
			if err == nil {
				err = errCompletion
			}
			return 0, err
		default:
		}
		if err := pump(ctx); err != nil {
			select {
			case value := <-done:
				return value, nil
			case herr := <-failed:
				if herr == nil {
					herr = errCompletion
				}
				return 0, herr
			default:
				return 0, err
			}
		}
	}
}
