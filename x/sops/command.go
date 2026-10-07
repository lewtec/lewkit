package sops

import (
	"context"
	"fmt"
	"os"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// Command runs Args with Env merged onto the process environment.
// An empty Target runs Args directly. A conda target runs them with conda run.
type Command struct {
	Env    Env
	Target Target
	Args   []string
}

// Run starts the command and waits. Stdin, stdout, and stderr are the process streams.
// A non-zero child status is returned as the process exit error.
func (c Command) Run(ctx context.Context) error {
	if len(c.Args) == 0 {
		return fmt.Errorf("sops: command is empty")
	}
	name, args, err := c.Target.Command(c.Args)
	if err != nil {
		return err
	}
	cmd, err := execdriver.Command(name, args...)
	if err != nil {
		return err
	}
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
