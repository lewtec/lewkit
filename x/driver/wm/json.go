package wm

import (
	"context"
	"encoding/json"
	"fmt"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// JSONViaCmd runs name with args and unmarshals stdout as JSON T.
func JSONViaCmd[T any](ctx context.Context, name string, args ...string) (T, error) {
	var zero T
	out, err := execdriver.MustRun(ctx, name, args...).Output()
	if err != nil {
		return zero, fmt.Errorf("%w: %w", ErrIPC, err)
	}
	var v T
	if err := json.Unmarshal(out, &v); err != nil {
		return zero, fmt.Errorf("%w: %w", ErrIPC, err)
	}
	return v, nil
}
