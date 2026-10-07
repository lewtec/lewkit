package sops

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// Command runs Args with Env merged onto the process environment.
type Command struct {
	Env  Env
	Args []string
}

// Run starts the command and waits. The exec driver is the one selected for ctx.
// Stdin, stdout, and stderr are the process streams.
// A non-zero child status is returned as the process exit error.
func (c Command) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("sops: nil context")
	}
	if len(c.Args) == 0 {
		return fmt.Errorf("sops: command is empty")
	}
	selected, err := driver.Get[execdriver.Driver](ctx)
	if err != nil {
		return err
	}
	cmd := selected.Command(c.Args[0], c.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = mergeEnviron(os.Environ(), c.Env)
	return execdriver.Run(ctx, cmd)
}

func mergeEnviron(base []string, overlay Env) []string {
	if len(overlay.keys) == 0 {
		return base
	}
	out := append([]string(nil), base...)
	index := make(map[string]int, len(out))
	for i, kv := range out {
		key, _, _ := strings.Cut(kv, "=")
		index[key] = i
	}
	for _, key := range overlay.keys {
		kv := key + "=" + overlay.vals[key]
		if i, ok := index[key]; ok {
			out[i] = kv
			continue
		}
		index[key] = len(out)
		out = append(out, kv)
	}
	return out
}
