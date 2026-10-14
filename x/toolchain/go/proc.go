package gotool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// spawn runs action in a new process. That process owns the compiler or
// linker tables and exits with them, so the caller does not reset or lock
// anything. x/workflow runs one spawn per package, and independent packages
// run together.
func spawn(ctx context.Context, action string, req childRequest) error {
	if ctx == nil {
		return ErrNilContext
	}
	f, err := os.CreateTemp("", "lewkit-toolchain-*.json")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := json.NewEncoder(f).Encode(req); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, exe)
	// GOOS and GOARCH are read by buildcfg when the child starts.
	// CGO_ENABLED stays off: this compiler has no cgo.
	env := dropEnv(os.Environ(), actionEnv, requestEnv)
	if req.GOOS != "" {
		env = append(dropEnv(env, "GOOS"), "GOOS="+req.GOOS)
	}
	if req.GOARCH != "" {
		env = append(dropEnv(env, "GOARCH"), "GOARCH="+req.GOARCH)
	}
	env = append(dropEnv(env, "CGO_ENABLED"), "CGO_ENABLED=0", actionEnv+"="+action, requestEnv+"="+name)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w%s: %w\n%s", ErrSpawn, action, err, out)
	}
	return nil
}

func dropEnv(env []string, keys ...string) []string {
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	out := make([]string, 0, len(env))
	for _, e := range env {
		k, _, _ := strings.Cut(e, "=")
		if drop[k] {
			continue
		}
		out = append(out, e)
	}
	return out
}
