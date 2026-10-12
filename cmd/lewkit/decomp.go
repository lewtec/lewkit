package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/lewtec/lewkit/cmd/lewkit/decompview"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/decomp"
	lewfs "github.com/lewtec/lewkit/x/fs"
)

var (
	// errProjectRequired means view was started without a sqlite program database.
	errProjectRequired = errors.New("missing --project")
	// errFunctionName means name was started without a function name.
	errFunctionName = errors.New("missing --name")
)

type decompCmd struct {
	project   cmd.StringArg `long:"project" default:"" help:"sqlite program database" ctx:""`
	summary   *decompSummaryCmd
	functions *decompFunctionsCmd
	listing   *decompListingCmd
	stream    *decompStreamCmd
	xrefs     *decompXRefsCmd
	strings   *decompStringsCmd
	match     *decompMatchCmd
	explain   *decompExplainCmd
	name      *decompNameCmd
	note      *decompNoteCmd
	signature *decompSignatureCmd
	view      *decompViewCmd
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

func (command *decompSummaryCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	_, err = io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), artifact.Summary())
	return err
}

type decompFunctionsCmd struct {
	path cmd.StringArg `help:"elf, pe, or macho path"`
}

func (decompFunctionsCmd) Description() string {
	return "list recovered functions"
}

func (command *decompFunctionsCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
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

func (command *decompListingCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	listing, err := artifact.Listing(commandAddress(command.at, artifact))
	if err != nil {
		return err
	}
	_, err = io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), listing)
	return err
}

type decompStreamCmd struct {
	path cmd.StringArg `help:"elf, pe, or macho path"`
}

func (decompStreamCmd) Description() string {
	return "print instructions and comments in address order"
}

func (command *decompStreamCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	_, err = io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), artifact.Stream())
	return err
}

type decompXRefsCmd struct {
	path cmd.StringArg      `help:"elf, pe, or macho path"`
	at   cmd.IntArg[uint64] `long:"at" default:"0" help:"address"`
}

func (decompXRefsCmd) Description() string {
	return "list references to one address"
}

func (command *decompXRefsCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	out := lewfs.ContextWriter(ctx, os.Stdout)
	for _, reference := range artifact.XRefs(command.at.Value()) {
		if _, err := fmt.Fprintf(out, "0x%x %s\n", reference.From, reference.Kind); err != nil {
			return err
		}
	}
	return nil
}

type decompStringsCmd struct {
	path cmd.StringArg `help:"elf, pe, or macho path"`
}

func (decompStringsCmd) Description() string {
	return "list strings in the image"
}

func (command *decompStringsCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	out := lewfs.ContextWriter(ctx, os.Stdout)
	for _, item := range artifact.Strings() {
		if _, err := fmt.Fprintf(out, "0x%x %s\n", item.Address, item.Text); err != nil {
			return err
		}
	}
	return nil
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

func (command *decompMatchCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
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
	result, err := artifact.Match(ctx, commandAddress(command.at, artifact), decomp.MatchConfig{Rounds: command.rounds.Value()})
	if result.Rounds > 0 {
		if writeErr := writeMatch(ctx, result); writeErr != nil {
			return writeErr
		}
	}
	return err
}

type decompExplainCmd struct {
	path    cmd.StringArg      `help:"elf, pe, or macho path"`
	at      cmd.IntArg[uint64] `long:"at" default:"0" help:"function address"`
	harness cmd.StringArg      `long:"harness" default:"grok" help:"grok executable for the signed-in subscription"`
	model   cmd.StringArg      `long:"model" default:"" help:"harness model; empty uses the signed-in default"`
	rounds  cmd.IntArg[int]    `long:"rounds" default:"1" help:"explain attempts"`
}

func (decompExplainCmd) Description() string {
	return "ask an attached model to explain one function"
}

func (command *decompExplainCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	artifact.Attach(decomp.Harness{
		Path:   command.harness.Value(),
		Model:  command.model.Value(),
		Schema: decomp.ExplainSchema,
	})
	result, err := artifact.Explain(ctx, commandAddress(command.at, artifact), decomp.MatchConfig{Rounds: command.rounds.Value()})
	if result.Rounds > 0 && result.Source != "" {
		if _, writeErr := io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), result.Source); writeErr != nil {
			return writeErr
		}
		if result.Source[len(result.Source)-1] != '\n' {
			if _, writeErr := io.WriteString(lewfs.ContextWriter(ctx, os.Stdout), "\n"); writeErr != nil {
				return writeErr
			}
		}
	}
	return err
}

type decompNameCmd struct {
	path   cmd.StringArg      `help:"elf, pe, or macho path"`
	at     cmd.IntArg[uint64] `long:"at" default:"0" help:"function address"`
	rename cmd.StringArg      `long:"name" default:"" help:"function name"`
}

func (decompNameCmd) Description() string {
	return "name one function"
}

func (command *decompNameCmd) Run(ctx context.Context) (err error) {
	if !command.rename.ArgSet() || command.rename.Value() == "" {
		return fmt.Errorf("decomp name: %w", errFunctionName)
	}
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	if err := artifact.SetName(ctx, command.at.Value(), command.rename.Value()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(lewfs.ContextWriter(ctx, os.Stdout), "%s\n", command.rename.Value())
	return err
}

type decompNoteCmd struct {
	path cmd.StringArg      `help:"elf, pe, or macho path"`
	at   cmd.IntArg[uint64] `long:"at" default:"0" help:"address"`
	text cmd.StringArg      `long:"text" default:"" help:"what this address means; empty clears the note"`
}

func (decompNoteCmd) Description() string {
	return "store what an address means"
}

func (command *decompNoteCmd) Run(ctx context.Context) (err error) {
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	if !command.at.ArgSet() && !command.text.ArgSet() {
		out := lewfs.ContextWriter(ctx, os.Stdout)
		for _, note := range artifact.Notes() {
			if _, err := fmt.Fprintf(out, "0x%x %s\n", note.Address, note.Text); err != nil {
				return err
			}
		}
		return nil
	}
	address := command.at.Value()
	if command.text.ArgSet() {
		if err := artifact.SetNote(ctx, address, command.text.Value()); err != nil {
			return err
		}
	}
	text, ok := artifact.Note(address)
	if !ok {
		if command.text.ArgSet() {
			return nil
		}
		return fmt.Errorf("no note at 0x%x", address)
	}
	_, err = fmt.Fprintf(lewfs.ContextWriter(ctx, os.Stdout), "%s\n", text)
	return err
}

type decompSignatureCmd struct {
	path cmd.StringArg      `help:"elf, pe, or macho path"`
	at   cmd.IntArg[uint64] `long:"at" default:"0" help:"function address"`
	text cmd.StringArg      `long:"text" default:"" help:"function signature"`
}

func (decompSignatureCmd) Description() string {
	return "store a function signature"
}

func (command *decompSignatureCmd) Run(ctx context.Context) (err error) {
	if !command.text.ArgSet() {
		return fmt.Errorf("decomp signature: missing --text")
	}
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	if err := artifact.SetSignature(ctx, command.at.Value(), command.text.Value()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(lewfs.ContextWriter(ctx, os.Stdout), "%s\n", command.text.Value())
	return err
}

type decompViewCmd struct {
	listen cmd.StringArg `long:"listen" default:"127.0.0.1:8766" help:"preview address"`
}

func (decompViewCmd) Description() string {
	return "preview the program database in the browser"
}

func (command *decompViewCmd) Run(ctx context.Context) (err error) {
	path := cmd.Get[string](ctx, "project")
	if path == "" {
		return fmt.Errorf("decomp view: %w", errProjectRequired)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("project: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("project %s is a directory", path)
	}
	project, err := decomp.Open(ctx, path)
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()

	listener, err := net.Listen("tcp", command.listen.Value())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if _, err := fmt.Fprintf(lewfs.ContextWriter(ctx, os.Stdout), "http://%s/\n", listener.Addr().String()); err != nil {
		_ = listener.Close()
		return err
	}
	return decompview.Serve(ctx, listener, decompview.Handler(project))
}

func openDecomp(ctx context.Context, path string) (*decomp.Project, *decomp.Artifact, error) {
	var project *decomp.Project
	if projectPath := cmd.Get[string](ctx, "project"); projectPath != "" {
		opened, err := decomp.Open(ctx, projectPath)
		if err != nil {
			return nil, nil, err
		}
		project = opened
	} else {
		project = decomp.New()
	}
	if path == "" || path == "-" {
		if artifact, ok := project.Artifact("stdin"); ok {
			return project, artifact, nil
		}
		data, err := readInput(ctx, path)
		if err != nil {
			_ = project.Close()
			return nil, nil, err
		}
		artifact, err := project.AddBytes(ctx, "stdin", data)
		if err != nil {
			_ = project.Close()
			return nil, nil, err
		}
		return project, artifact, nil
	}
	data, err := readInput(ctx, path)
	if err != nil {
		if artifact, ok := project.ArtifactFile(path); ok {
			return project, artifact, nil
		}
		_ = project.Close()
		return nil, nil, err
	}
	artifact, err := project.Remember(ctx, path, data)
	if err != nil {
		_ = project.Close()
		return nil, nil, err
	}
	return project, artifact, nil
}

func commandAddress(at cmd.IntArg[uint64], artifact *decomp.Artifact) uint64 {
	if !at.ArgSet() {
		return artifact.Entry()
	}
	return at.Value()
}

func joinClose(err, closeErr error) error {
	if err != nil {
		return err
	}
	return closeErr
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
