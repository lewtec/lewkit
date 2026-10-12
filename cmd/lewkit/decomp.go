package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/decomp"
	lewfs "github.com/lewtec/lewkit/x/fs"
)

type decompCmd struct {
	summary   *decompSummaryCmd
	functions *decompFunctionsCmd
	listing   *decompListingCmd
	match     *decompMatchCmd
}

func (decompCmd) Description() string {
	return "load a native artifact and match a decompilation"
}

type decompSummaryCmd struct {
	path cmd.StringArg `help:"elf, pe, or macho path"`
}

func (decompSummaryCmd) Description() string {
	return "summarize one native artifact"
}

func (command *decompSummaryCmd) Run(ctx context.Context) error {
	artifact, err := openArtifact(ctx, command.path.Value())
	if err != nil {
		return err
	}
	_, err = io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), artifact.Summary())
	return err
}

type decompFunctionsCmd struct {
	path cmd.StringArg `help:"elf, pe, or macho path"`
}

func (decompFunctionsCmd) Description() string {
	return "list recovered functions"
}

func (command *decompFunctionsCmd) Run(ctx context.Context) error {
	artifact, err := openArtifact(ctx, command.path.Value())
	if err != nil {
		return err
	}
	out := lewfs.ContextWriter(ctx, os.Stdout)
	for _, function := range artifact.Functions() {
		if _, err := fmt.Fprintf(out, "0x%x\t%s\t%d\n", function.Address, function.Name, function.Size); err != nil {
			return err
		}
	}
	return nil
}

type decompListingCmd struct {
	path cmd.StringArg      `help:"elf, pe, or macho path"`
	at   cmd.IntArg[uint64] `long:"at" default:"0" help:"function address"`
}

func (decompListingCmd) Description() string {
	return "print blocks and instructions for one function"
}

func (command *decompListingCmd) Run(ctx context.Context) error {
	artifact, err := openArtifact(ctx, command.path.Value())
	if err != nil {
		return err
	}
	listing, err := artifact.Listing(command.at.Value())
	if err != nil {
		return err
	}
	_, err = io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), listing)
	return err
}

type decompMatchCmd struct {
	path     cmd.StringArg      `help:"elf, pe, or macho path"`
	at       cmd.IntArg[uint64] `long:"at" default:"0" help:"function address"`
	harness  cmd.StringArg      `long:"harness" default:"grok" help:"grok executable for the signed-in subscription"`
	model    cmd.StringArg      `long:"model" default:"" help:"harness model; empty uses the signed-in default"`
	rounds   cmd.IntArg[int]    `long:"rounds" default:"4" help:"match attempts"`
	endpoint cmd.StringArg      `long:"endpoint" default:"" help:"chat completions URL; skips the harness"`
	apiKey   cmd.StringArg      `long:"api-key" env:"XAI_API_KEY" default:"" help:"bearer token used with --endpoint"`
}

func (decompMatchCmd) Description() string {
	return "ask an attached model to match-decompile one function"
}

func (command *decompMatchCmd) Run(ctx context.Context) error {
	artifact, err := openArtifact(ctx, command.path.Value())
	if err != nil {
		return err
	}
	if command.endpoint.Value() != "" {
		model := command.model.Value()
		if model == "" {
			model = "grok-4"
		}
		artifact.Attach(decomp.Chat{
			Endpoint: command.endpoint.Value(),
			Model:    model,
			APIKey:   command.apiKey.Value(),
		})
	} else {
		artifact.Attach(decomp.Harness{
			Path:  command.harness.Value(),
			Model: command.model.Value(),
		})
	}
	result, err := artifact.Match(ctx, command.at.Value(), decomp.MatchConfig{Rounds: command.rounds.Value()})
	if result.Rounds > 0 {
		if writeErr := writeMatch(ctx, result); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func openArtifact(ctx context.Context, path string) (*decomp.Artifact, error) {
	data, err := readInput(ctx, path)
	if err != nil {
		return nil, err
	}
	name := path
	if name == "" || name == "-" {
		name = "stdin"
	} else {
		name = filepath.Base(name)
	}
	return decomp.New().AddBytes(ctx, name, data)
}

func writeMatch(ctx context.Context, result decomp.Result) error {
	out := lewfs.ContextWriter(ctx, os.Stdout)
	if result.Matched {
		if _, err := fmt.Fprintf(out, "matched in %d\n", result.Rounds); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(out, "unmatched after %d\n", result.Rounds); err != nil {
		return err
	}
	if result.Name != "" {
		if _, err := fmt.Fprintf(out, "%s\n", result.Name); err != nil {
			return err
		}
	}
	if result.Signature != "" {
		if _, err := fmt.Fprintf(out, "%s\n", result.Signature); err != nil {
			return err
		}
	}
	if result.Source != "" {
		if _, err := io.WriteString(out, result.Source); err != nil {
			return err
		}
		if result.Source[len(result.Source)-1] != '\n' {
			if _, err := io.WriteString(out, "\n"); err != nil {
				return err
			}
		}
	}
	for _, note := range result.Residuals {
		if _, err := fmt.Fprintf(out, "- %s\n", note); err != nil {
			return err
		}
	}
	return nil
}
