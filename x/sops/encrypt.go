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
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/tool"
	_ "github.com/lewtec/lewkit/x/tool/github"
)

var (
	ErrEncryptErrSops        = errors.New("sops")
	ErrEncryptSops           = errors.New("sops")
	ErrFileParse             = errors.New("")
	ErrSopsCommandIsEmpty    = errs.Wrap(errs.ErrEmpty, "sops: command is empty")
	ErrSopsEmptyDotenvKey    = errs.Wrap(errs.ErrEmpty, "sops: empty dotenv key")
	ErrSopsInvalidDotenvLine = errs.Wrap(errs.ErrInvalid, "sops: invalid dotenv line")
)

// sopsRef is the lazy tool ref. The version lives in modot.lock.json.
const sopsRef = "github:getsops/sops"

// ErrNoConfig means no .sops.yaml was found above the output path.
var ErrNoConfig = errs.Wrap(errs.ErrNotFound, "sops config file not found")

// Encrypt encrypts plaintext for path with the sops lazy tool.
// The version is the modot.lock.json pin above path, or latest when
// that pin is absent. x/tool installs the binary. The creation rule
// is the one that sops reads from .sops.yaml. A missing config file
// is ErrNoConfig.
func Encrypt(ctx context.Context, path string, plaintext []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir := commandDir(filepath.Dir(abs))
	bin, err := ensureSops(ctx, filepath.Dir(abs))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncryptSops, err)
	}
	selected, err := driver.Get[execdriver.Driver](ctx)
	if err != nil {
		return nil, err
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

func ensureSops(ctx context.Context, directory string) (string, error) {
	spec, err := sopsSpec(directory)
	if err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	store, err := tool.Open(filepath.Join(cache, release.Name(), "tools"))
	if err != nil {
		return "", err
	}
	return store.Ensure(ctx, spec, "sops")
}

func sopsSpec(directory string) (string, error) {
	return tool.LockSpec(directory, sopsRef)
}

func commandDir(dir string) string {
	for {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

func encryptErr(stderr string, err error) error {
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = err.Error()
	}
	if strings.Contains(msg, "config file not found") || strings.Contains(msg, ".sops.yaml") {
		return fmt.Errorf("%w: %s: %w", ErrNoConfig, msg, err)
	}
	return fmt.Errorf("%w: %s: %w", ErrEncryptErrSops, msg, err)
}
