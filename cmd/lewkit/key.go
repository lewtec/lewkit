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
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/sops"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/ui/tui"
)

var (
	errNameRequired   = errors.New("publisher name is required")
	errPathRequired   = errors.New("PKCS#12 path is required")
	errPathIsDir      = errors.New("PKCS#12 path is a directory")
	errReplaceRefused = errors.New("PKCS#12 already exists")
)

// keyCmd writes one passwordless RSA-2048 PKCS#12 and encrypts it with the
// sops config above that path. A missing name, path, or replace is an
// interview on the caller context. A missing .sops.yaml is an error.
type keyCmd struct {
	name  cmd.StringArg `long:"name" help:"certificate common name" default:""`
	out   cmd.StringArg `long:"out" env:"LEWKIT_SIGN_P12" help:"PKCS#12 file to write" default:""`
	force cmd.Flag      `long:"force" help:"replace an existing PKCS#12"`
}

func (keyCmd) Description() string {
	return "write an RSA-2048 PKCS#12 publisher key"
}

func (c *keyCmd) Run(ctx context.Context) error {
	// The progress view stops after Run returns. Interview once the terminal is back.
	if taskgroup.FromContext(ctx) != nil {
		entry.After(c.write)
		return nil
	}
	return c.write(ctx)
}

func (c *keyCmd) write(ctx context.Context) error {
	return c.generate(ctx, tui.Run)
}

type interview func(context.Context, []tui.Question) ([]tui.Answer, error)

func (c *keyCmd) generate(ctx context.Context, run interview) error {
	if run == nil {
		run = tui.Run
	}
	name, path, err := c.interview(ctx, run)
	if err != nil {
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

func (c *keyCmd) interview(ctx context.Context, run interview) (string, string, error) {
	name := strings.TrimSpace(c.name.Value())
	path := strings.TrimSpace(c.out.Value())
	questions := make([]tui.Question, 0, 3)
	if name == "" {
		questions = append(questions, tui.Question{Prompt: "Publisher name"})
	}
	if path == "" {
		questions = append(questions, tui.Question{Prompt: "PKCS#12 path"})
	}
	confirmed := ""
	if path != "" {
		exists, err := occupied(path)
		if err != nil {
			return "", "", err
		}
		if exists && !c.force.Value() {
			confirmed = path
			questions = append(questions, tui.Question{Prompt: "Replace " + path + "?", Confirm: true})
		}
	}
	if len(questions) > 0 {
		answers, err := run(ctx, questions)
		if err != nil {
			return "", "", err
		}
		if len(answers) != len(questions) {
			return "", "", tui.ErrCanceled
		}
		n := 0
		if name == "" {
			name = strings.TrimSpace(answers[n].Text)
			n++
			if name == "" {
				return "", "", errNameRequired
			}
		}
		if path == "" {
			path = strings.TrimSpace(answers[n].Text)
			n++
			if path == "" {
				return "", "", errPathRequired
			}
		}
		if confirmed != "" && !answers[n].Yes {
			return "", "", errReplaceRefused
		}
	}
	if name == "" {
		return "", "", errNameRequired
	}
	if path == "" {
		return "", "", errPathRequired
	}
	if path == confirmed {
		return name, path, nil
	}
	exists, err := occupied(path)
	if err != nil {
		return "", "", err
	}
	if !exists || c.force.Value() {
		return name, path, nil
	}
	answers, err := run(ctx, []tui.Question{{
		Prompt:  "Replace " + path + "?",
		Confirm: true,
	}})
	if err != nil {
		return "", "", err
	}
	if len(answers) != 1 || !answers[0].Yes {
		return "", "", errReplaceRefused
	}
	return name, path, nil
}

func occupied(path string) (bool, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, errPathIsDir
	}
	return true, nil
}
