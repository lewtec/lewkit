package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/sops"
	"github.com/lewtec/lewkit/x/taskgroup"
)

var (
	errNameRequired   = errors.New("publisher name is required")
	errPathRequired   = errors.New("PKCS#12 path is required")
	errReplaceRefused = errors.New("PKCS#12 already exists")
)

// keyCmd writes one passwordless RSA-2048 PKCS#12 and encrypts it with the
// sops config above that path. A name or path that was not supplied is
// prompted on the caller context. A missing .sops.yaml is an error.
type keyCmd struct {
	name  cmd.StringArg `long:"name" help:"certificate common name" default:""`
	out   cmd.StringArg `long:"out" env:"LEWKIT_SIGN_P12" help:"PKCS#12 file to write" default:""`
	force cmd.Flag      `long:"force" help:"replace an existing PKCS#12"`
}

func (keyCmd) Description() string {
	return "write an RSA-2048 PKCS#12 publisher key"
}

func (c *keyCmd) Run(ctx context.Context) error {
	// The progress view stops after Run returns. Prompt once the terminal is back.
	if taskgroup.FromContext(ctx) != nil {
		entry.After(c.write)
		return nil
	}
	return c.write(ctx)
}

func (c *keyCmd) write(ctx context.Context) error {
	return c.generate(ctx, launcher.Prompt, launcher.Confirm)
}

func (c *keyCmd) generate(ctx context.Context, ask func(context.Context, string) (string, error), confirm func(context.Context, string) (bool, error)) error {
	name, err := c.publisher(ctx, ask)
	if err != nil {
		return err
	}
	path, err := c.destination(ctx, ask)
	if err != nil {
		return err
	}
	if err := c.replace(ctx, path, confirm); err != nil {
		return err
	}
	id, err := sign.Generate(name)
	if err != nil {
		return err
	}
	der, err := id.PKCS12("")
	if err != nil {
		return err
	}
	enc, err := sops.Encrypt(ctx, path, der)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, enc, 0o600); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, path)
	return nil
}

func (c *keyCmd) publisher(ctx context.Context, ask func(context.Context, string) (string, error)) (string, error) {
	name := strings.TrimSpace(c.name.Value())
	if name != "" {
		return name, nil
	}
	answered, err := ask(ctx, "Publisher name")
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(answered)
	if name == "" {
		return "", errNameRequired
	}
	return name, nil
}

func (c *keyCmd) destination(ctx context.Context, ask func(context.Context, string) (string, error)) (string, error) {
	path := strings.TrimSpace(c.out.Value())
	if path != "" {
		return path, nil
	}
	answered, err := ask(ctx, "PKCS#12 path")
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(answered)
	if path == "" {
		return "", errPathRequired
	}
	return path, nil
}

func (c *keyCmd) replace(ctx context.Context, path string, confirm func(context.Context, string) (bool, error)) error {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("PKCS#12 path is a directory")
	}
	if c.force.Value() {
		return nil
	}
	ok, err := confirm(ctx, "Replace "+path+"?")
	if err != nil {
		return err
	}
	if !ok {
		return errReplaceRefused
	}
	return nil
}
