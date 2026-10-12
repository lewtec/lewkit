package decomp

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/disasm"
)

var (
	// ErrNoCode is [Project.Add] on an image with no executable section.
	ErrNoCode = errors.New("no executable section")
	// ErrArtifact is [Project.Add] when that name is already open.
	ErrArtifact = errors.New("artifact already open")
	// ErrNoModel is [Artifact.Match] before [Project.Attach].
	ErrNoModel = errors.New("no model attached")
	// ErrNotFound is a function address that is not a function start.
	ErrNotFound = errors.New("function not found")
	// ErrUnmatched is [Artifact.Match] when the rounds are spent.
	ErrUnmatched = errors.New("decompilation does not match")
)

// Project is one session: native artifacts and, once attached, a model.
// [Open] also keeps those artifacts in a sqlite file. [New] keeps them in memory.
// A project is not safe for concurrent use.
type Project struct {
	model     Model
	artifacts []*Artifact
	db        *sql.DB
	path      string
}

// Artifact is one loaded image and the names, comments, and matched
// source written onto it.
type Artifact struct {
	project      *Project
	name         string
	format       string
	architecture disasm.Architecture
	mode         disasm.Mode
	entry        uint64
	sections     []disasm.Section
	symbols      map[uint64]string
	functions    []Function
	byAddress    map[uint64]*Function
}

// Function is one recovered function.
type Function struct {
	Address      uint64
	Name         string
	Signature    string
	Comment      string
	Source       string
	Size         uint64
	Blocks       []Block
	Instructions []disasm.Instruction
	References   []Reference
	Strings      []string
}

// Block is a straight run of instructions with one control-flow kind.
type Block struct {
	Address      uint64
	Kind         string
	Target       uint64
	Fall         uint64
	Instructions []disasm.Instruction
}

// Reference is a call, branch, jump, or data address used by an instruction.
type Reference struct {
	From uint64
	To   uint64
	Kind string
}

// New starts an empty session.
func New() *Project {
	return &Project{}
}

// Attach sets the model used by [Artifact.Match]. A later call replaces it.
func (project *Project) Attach(model Model) {
	project.model = model
}

// Attach sets the model on the session that owns this artifact.
func (artifact *Artifact) Attach(model Model) {
	if artifact.project != nil {
		artifact.project.Attach(model)
	}
}

// Add loads one ELF, PE, or Mach-O image under name.
func (project *Project) Add(ctx context.Context, name string, reader io.ReaderAt, size int64) (*Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("empty artifact name")
	}
	for _, artifact := range project.artifacts {
		if artifact.name == name {
			return nil, fmt.Errorf("%w: %s", ErrArtifact, name)
		}
	}
	image, err := readImage(reader, size)
	if err != nil {
		return nil, err
	}
	object, format, err := disasm.OpenObject(bytes.NewReader(image), int64(len(image)))
	if err != nil {
		return nil, err
	}
	defer object.Close()
	artifact, err := lift(ctx, object, format)
	if err != nil {
		return nil, err
	}
	artifact.project = project
	artifact.name = name
	if err := project.insertArtifact(ctx, artifact, image); err != nil {
		return nil, err
	}
	project.artifacts = append(project.artifacts, artifact)
	return artifact, nil
}

func readImage(reader io.ReaderAt, size int64) ([]byte, error) {
	if size < 0 {
		return nil, errors.New("negative artifact size")
	}
	image := make([]byte, size)
	if size == 0 {
		return image, nil
	}
	read, err := reader.ReadAt(image, 0)
	if read == len(image) {
		return image, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, io.ErrUnexpectedEOF
}

// AddBytes loads an image held in memory.
func (project *Project) AddBytes(ctx context.Context, name string, data []byte) (*Artifact, error) {
	return project.Add(ctx, name, bytes.NewReader(data), int64(len(data)))
}

// Artifacts returns the open images in the order they were added.
func (project *Project) Artifacts() []*Artifact {
	out := make([]*Artifact, len(project.artifacts))
	copy(out, project.artifacts)
	return out
}

// Name is the name passed to [Project.Add].
func (artifact *Artifact) Name() string { return artifact.name }

// Format is elf, pe, or macho.
func (artifact *Artifact) Format() string { return artifact.format }

// Entry is the image entry address, or 0 when the format has none.
func (artifact *Artifact) Entry() uint64 { return artifact.entry }

// Architecture is the Capstone architecture of the image.
func (artifact *Artifact) Architecture() disasm.Architecture { return artifact.architecture }

// Functions returns every recovered function, lowest address first.
func (artifact *Artifact) Functions() []Function {
	out := make([]Function, len(artifact.functions))
	for index, function := range artifact.functions {
		out[index] = cloneFunction(function)
	}
	return out
}

// Function returns a copy of the function that starts at address.
func (artifact *Artifact) Function(address uint64) (Function, bool) {
	function := artifact.byAddress[address]
	if function == nil {
		return Function{}, false
	}
	return cloneFunction(*function), true
}

// SetName renames the function that starts at address.
// An [Open] project writes the new name into the sqlite file.
func (artifact *Artifact) SetName(ctx context.Context, address uint64, name string) error {
	if name == "" {
		return errors.New("empty function name")
	}
	function := artifact.byAddress[address]
	if function == nil {
		return fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	previous := function.Name
	function.Name = name
	if err := artifact.saveFunction(ctx, function); err != nil {
		function.Name = previous
		return err
	}
	return nil
}

// SetSignature records a prototype for the function that starts at address.
// An [Open] project writes the prototype into the sqlite file.
func (artifact *Artifact) SetSignature(ctx context.Context, address uint64, signature string) error {
	function := artifact.byAddress[address]
	if function == nil {
		return fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	previous := function.Signature
	function.Signature = signature
	if err := artifact.saveFunction(ctx, function); err != nil {
		function.Signature = previous
		return err
	}
	return nil
}

// SetComment records a note on the function that starts at address.
// An [Open] project writes the note into the sqlite file.
func (artifact *Artifact) SetComment(ctx context.Context, address uint64, comment string) error {
	function := artifact.byAddress[address]
	if function == nil {
		return fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	previous := function.Comment
	function.Comment = comment
	if err := artifact.saveFunction(ctx, function); err != nil {
		function.Comment = previous
		return err
	}
	return nil
}

func cloneFunction(function Function) Function {
	function.Blocks = append([]Block(nil), function.Blocks...)
	for index := range function.Blocks {
		function.Blocks[index].Instructions = cloneInstructions(function.Blocks[index].Instructions)
	}
	function.Instructions = cloneInstructions(function.Instructions)
	function.References = append([]Reference(nil), function.References...)
	function.Strings = append([]string(nil), function.Strings...)
	return function
}

func cloneInstructions(instructions []disasm.Instruction) []disasm.Instruction {
	out := append([]disasm.Instruction(nil), instructions...)
	for index := range out {
		out[index].Bytes = append([]byte(nil), out[index].Bytes...)
	}
	return out
}
