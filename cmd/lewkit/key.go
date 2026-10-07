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
	errNameRequired     = errors.New("publisher name is required")
	errPathRequired     = errors.New("PKCS#12 path is required")
	errPasswordMismatch = errors.New("PKCS#12 passwords do not match")
	errReplaceRefused   = errors.New("PKCS#12 already exists")
)

// keyCmd writes one RSA-2048 PKCS#12. A name, path, or password that was
// not supplied is prompted on the caller context.
type keyCmd struct {
	name     cmd.StringArg `long:"name" help:"certificate common name" default:""`
	out      cmd.StringArg `long:"out" env:"LEWKIT_SIGN_P12" help:"PKCS#12 file to write" default:""`
	password sops.File     `long:"p12-password" env:"LEWKIT_SIGN_P12_PASSWORD" help:"PKCS#12 password file. Prompted when omitted. A SOPS age file is decrypted." default:""`
	force    cmd.Flag      `long:"force" help:"replace an existing PKCS#12"`
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
	password, err := c.secret(ctx, ask)
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
	der, err := id.PKCS12(password)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, der, 0o600); err != nil {
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

func (c *keyCmd) secret(ctx context.Context, ask func(context.Context, string) (string, error)) (string, error) {
	if c.password.Value() != nil {
		return c.password.Text(), nil
	}
	password, err := ask(ctx, "PKCS#12 password")
	if err != nil {
		return "", err
	}
	again, err := ask(ctx, "Repeat PKCS#12 password")
	if err != nil {
		return "", err
	}
	if password != again {
		return "", errPasswordMismatch
	}
	return password, nil
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
