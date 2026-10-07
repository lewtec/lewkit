package sops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// ErrNoConfig means sops found no .sops.yaml for the output path.
var ErrNoConfig = errors.New("sops config file not found")

// Encrypt runs sops encrypt. The creation rule is the one .sops.yaml
// selects for path. A missing config file is ErrNoConfig.
func Encrypt(ctx context.Context, path string, plaintext []byte) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	selected, err := driver.Get[execdriver.Driver](ctx)
	if err != nil {
		return nil, err
	}
	bin, err := selected.Which(ctx, "sops")
	if err != nil {
		return nil, fmt.Errorf("sops: %w", err)
	}
	cmd := selected.Command(bin, "encrypt",
		"--input-type", "binary",
		"--output-type", "json",
		"--filename-override", abs,
	)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(plaintext)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := execdriver.Output(ctx, cmd)
	if err != nil {
		return nil, encryptErr(stderr.String(), err)
	}
	return out, nil
}

func encryptErr(stderr string, err error) error {
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = err.Error()
	}
	if strings.Contains(msg, "config file not found") {
		return fmt.Errorf("%w: %s", ErrNoConfig, msg)
	}
	return fmt.Errorf("sops: %s", msg)
}
