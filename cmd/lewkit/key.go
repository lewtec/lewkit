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
	out   cmd.StringArg `long:"out" env:"LEWKIT_SIGN_P12" help:"PKCS#12 file to write" default:"publisher.p12"`
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
	return c.generate(ctx, func(ctx context.Context, questions []tui.Question, grow tui.Grow) ([]tui.Answer, error) {
		return tui.Run(ctx, tui.New(questions).WithGrow(grow))
	})
}

type interview func(context.Context, []tui.Question, tui.Grow) ([]tui.Answer, error)

func (c *keyCmd) generate(ctx context.Context, run interview) error {
	if run == nil {
		run = func(ctx context.Context, questions []tui.Question, grow tui.Grow) ([]tui.Answer, error) {
			return tui.Run(ctx, tui.New(questions).WithGrow(grow))
		}
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
	asks, err := cmd.Asks(c)
	if err != nil {
		return "", "", err
	}
	questions := make([]tui.Question, len(asks))
	for i, ask := range asks {
		questions[i] = tui.Question{Prompt: ask.Prompt, Default: ask.Default, Choices: ask.Choices}
	}
	var grow tui.Grow
	if !c.out.ArgSet() || strings.TrimSpace(c.out.Value()) == "" {
		grow = c.replaceGrow(asks)
	} else if !c.force.Value() {
		exists, err := occupied(c.out.Value())
		if err != nil {
			return "", "", err
		}
		if exists {
			questions = append(questions, replaceQuestion(c.out.Value()))
		}
	}
	var answers []tui.Answer
	if len(questions) > 0 {
		answers, err = run(ctx, questions, grow)
		if err != nil {
			return "", "", err
		}
		if len(answers) < len(questions) {
			return "", "", tui.ErrCanceled
		}
	}
	for i, ask := range asks {
		if err := ask.Apply(strings.TrimSpace(answers[i].Text)); err != nil {
			return "", "", err
		}
	}
	name := strings.TrimSpace(c.name.Value())
	path := strings.TrimSpace(c.out.Value())
	if name == "" {
		return "", "", errNameRequired
	}
	if path == "" {
		return "", "", errPathRequired
	}
	if c.force.Value() {
		return name, path, nil
	}
	if len(answers) > len(asks) {
		if answers[len(asks)].Text != "yes" {
			return "", "", errReplaceRefused
		}
		return name, path, nil
	}
	exists, err := occupied(path)
	if err != nil {
		return "", "", err
	}
	if !exists {
		return name, path, nil
	}
	more, err := run(ctx, []tui.Question{replaceQuestion(path)}, nil)
	if err != nil {
		return "", "", err
	}
	if len(more) != 1 || more[0].Text != "yes" {
		return "", "", errReplaceRefused
	}
	return name, path, nil
}

func (c *keyCmd) replaceGrow(asks []cmd.Ask) tui.Grow {
	return func(answers []tui.Answer) ([]tui.Question, error) {
		if c.force.Value() {
			return nil, nil
		}
		path, ok := answerFor(asks, answers, "out")
		if !ok {
			path = strings.TrimSpace(c.out.Value())
		}
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, nil
		}
		exists, err := occupied(path)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, nil
		}
		return []tui.Question{replaceQuestion(path)}, nil
	}
}

func answerFor(asks []cmd.Ask, answers []tui.Answer, name string) (string, bool) {
	for i, ask := range asks {
		if ask.Name != name || i >= len(answers) {
			continue
		}
		return answers[i].Text, true
	}
	return "", false
}

func replaceQuestion(path string) tui.Question {
	return tui.Question{
		Prompt:  "Replace " + path + "?",
		Choices: []string{"yes", "no"},
		Default: "no",
	}
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
