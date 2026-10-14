package decomp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/lewtec/lewkit/cmd/lewkit/internal/input"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db"
	xdecomp "github.com/lewtec/lewkit/x/disasm/decomp"
	"github.com/lewtec/lewkit/x/disasm/decomp/view"
	"github.com/lewtec/lewkit/x/errs"
	lewfs "github.com/lewtec/lewkit/x/fs"
)

var (
	// errProjectRequired means view was started without a sqlite program database.
	errProjectRequired = errors.New("missing --project")
	// errFunctionName means name was started without a function name.
	errFunctionName = errors.New("missing --name")
	// errCandidate means check was started without the caller's JSON.
	errCandidate                  = errors.New("missing candidate")
	ErrDecompNoteCmdRunNoNoteAt0x = errs.Wrap(errs.ErrNotFound, "no note at 0x")
	ErrDecompSignatureMissingText = errs.Wrap(errs.ErrMissing, "decomp signature: missing --text")
	ErrDecompViewCmdRunListen     = errors.New("listen")
)

type Cmd struct {
	project   xdecomp.DBArg `long:"project" default:"" help:"sqlite program database" ctx:""`
	summary   *decompSummaryCmd
	functions *decompFunctionsCmd
	listing   *decompListingCmd
	stream    *decompStreamCmd
	xrefs     *decompXRefsCmd
	strings   *decompStringsCmd
	check     *decompCheckCmd
	name      *decompNameCmd
	note      *decompNoteCmd
	signature *decompSignatureCmd
	view      *decompViewCmd
}

func (Cmd) Description() string {
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

type decompCheckCmd struct {
	path cmd.StringArg      `help:"elf, pe, or macho path"`
	at   cmd.IntArg[uint64] `long:"at" default:"0" help:"function address"`
	text cmd.StringArg      `long:"text" default:"" help:"candidate JSON the caller already wrote"`
}

func (decompCheckCmd) Description() string {
	return "compare a candidate with one function"
}

func (command *decompCheckCmd) Run(ctx context.Context) (err error) {
	if !command.text.ArgSet() || command.text.Value() == "" {
		return fmt.Errorf("decomp check: %w", errCandidate)
	}
	project, artifact, err := openDecomp(ctx, command.path.Value())
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()
	result, err := artifact.Check(ctx, commandAddress(command.at, artifact), command.text.Value())
	if result.Matched || len(result.Residuals) > 0 {
		if writeErr := writeMatch(ctx, result); writeErr != nil {
			return writeErr
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
		return fmt.Errorf("%w%x", ErrDecompNoteCmdRunNoNoteAt0x, address)
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
		return ErrDecompSignatureMissingText
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
	conn, ok := cmd.Lookup[*db.Conn[xdecomp.Queries]](ctx, "project")
	if !ok || conn == nil || strings.TrimSpace(conn.URL()) == "" {
		return fmt.Errorf("decomp view: %w", errProjectRequired)
	}
	project, err := xdecomp.OpenConn(ctx, conn)
	if err != nil {
		return err
	}
	defer func() { err = joinClose(err, project.Close()) }()

	listener, err := net.Listen("tcp", command.listen.Value())
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDecompViewCmdRunListen, err)
	}
	if _, err := fmt.Fprintf(lewfs.ContextWriter(ctx, os.Stdout), "http://%s/\n", listener.Addr().String()); err != nil {
		_ = listener.Close()
		return err
	}
	return view.Serve(ctx, listener, view.Handler(project))
}

func openDecomp(ctx context.Context, path string) (*xdecomp.Project, *xdecomp.Artifact, error) {
	var project *xdecomp.Project
	if conn, ok := cmd.Lookup[*db.Conn[xdecomp.Queries]](ctx, "project"); ok && conn != nil && strings.TrimSpace(conn.URL()) != "" {
		opened, err := xdecomp.OpenConn(ctx, conn)
		if err != nil {
			return nil, nil, err
		}
		project = opened
	} else {
		project = xdecomp.New()
	}
	if path == "" || path == "-" {
		if artifact, ok := project.Artifact("stdin"); ok {
			return project, artifact, nil
		}
		data, err := input.Read(ctx, path)
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
	data, err := input.Read(ctx, path)
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

func commandAddress(at cmd.IntArg[uint64], artifact *xdecomp.Artifact) uint64 {
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

func writeMatch(ctx context.Context, result xdecomp.Result) error {
	out := lewfs.ContextWriter(ctx, os.Stdout)
	if result.Matched {
		if _, err := fmt.Fprintf(out, "matched\n"); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(out, "unmatched\n"); err != nil {
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
