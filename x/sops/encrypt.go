package sops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/tool"
	_ "github.com/lewtec/lewkit/x/tool/github"
)

// sopsRef is the lazy tool ref. The version lives in modot.lock.json.
const sopsRef = "github:getsops/sops"

// ErrNoConfig means no .sops.yaml was found above the output path.
var ErrNoConfig = errors.New("sops config file not found")

// Encrypt encrypts plaintext for path with the sops lazy tool.
// The version is the modot.lock.json pin above path, or latest when
// that pin is absent. x/tool installs the binary. The creation rule
// is the one that sops reads from .sops.yaml. A missing config file
// is ErrNoConfig.
func Encrypt(ctx context.Context, path string, plaintext []byte) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
		return nil, fmt.Errorf("sops: %w", err)
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
	for {
		spec, found, err := sopsSpecIn(directory)
		if err != nil {
			return "", err
		}
		if found {
			return spec, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return sopsRef + "@latest", nil
		}
		directory = parent
	}
}

func sopsSpecIn(directory string) (string, bool, error) {
	root, err := lewpath.Open(directory)
	if err != nil {
		return "", false, nil
	}
	defer root.Close()
	name := lewpath.New("modot.lock.json")
	isFile, err := name.IsFile(root)
	if err != nil || !isFile {
		return "", false, err
	}
	body, err := name.ReadFile(root)
	if err != nil {
		return "", false, err
	}
	var lock struct {
		Dependencies []struct {
			Kind         string `json:"kind"`
			Ref          string `json:"ref"`
			CurrentValue string `json:"currentValue"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return "", false, fmt.Errorf("modot.lock.json: %w", err)
	}
	for _, dependency := range lock.Dependencies {
		if dependency.Kind == "tool" && dependency.Ref == sopsRef && dependency.CurrentValue != "" {
			return sopsRef + "@" + dependency.CurrentValue, true, nil
		}
	}
	return "", false, nil
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
	return fmt.Errorf("sops: %s: %w", msg, err)
}
