package decomp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/db"
	"github.com/lewtec/lewkit/x/disasm"
	"github.com/lewtec/lewkit/x/disasm/decomp/schema"
	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrArtifactLoadBlocksLoadBlocks        = errors.New("load blocks")
	ErrArtifactLoadImageStringsLoadStrings = errors.New("load strings")
	ErrArtifactLoadNotesLoadNotes          = errors.New("load notes")
	ErrArtifactLoadStringsLoadStrings      = errors.New("load strings")
	ErrArtifactLoadSymbolsLoadSymbols      = errors.New("load symbols")
	ErrArtifactSaveNoteSaveNote0x          = errors.New("save note 0x")
	ErrBlockRunsPastItsInstructions        = errors.New("block runs past its instructions")
	ErrBlocksDoNotCoverItsInstructions     = errors.New("blocks do not cover its instructions")
	ErrCandidateHasAnUnfinishedJsonObject  = errors.New("candidate has an unfinished json object")
	ErrCandidateHasNoJsonObject            = errs.Wrap(errs.ErrNotFound, "candidate has no json object")
	ErrCloseProject                        = errors.New("close project")
	ErrEmptyArtifactName                   = errs.Wrap(errs.ErrEmpty, "empty artifact name")
	ErrEmptyFunctionName                   = errs.Wrap(errs.ErrEmpty, "empty function name")
	ErrInsertFunctionStoreBlock            = errors.New("store block")
	ErrInsertFunctionStoreString           = errors.New("store string")
	ErrLoadFunctions                       = errors.New("load functions")
	ErrLoadInstructions                    = errors.New("load instructions")
	ErrLoadReferences                      = errors.New("load references")
	ErrLoadSections                        = errors.New("load sections")
	ErrNegativeArtifactSize                = errors.New("negative artifact size")
	ErrOpenConnOpenProject                 = errors.New("open project")
	ErrOpenOpenProject                     = errors.New("open project")
	ErrOpenProjectEmptyPath                = errs.Wrap(errs.ErrEmpty, "open project: empty path")

	ErrParseAddressAddress              = errors.New("address")
	ErrParseCandidateCandidate          = errors.New("candidate")
	ErrProjectAdoptPathStorePath        = errors.New("store path")
	ErrProjectInsertArtifactStoreString = errors.New("store string")
	ErrProjectInsertArtifactStoreSymbol = errors.New("store symbol")
	ErrProjectLoadLoadProject           = errors.New("load project")
	ErrRejectForeignDBOpenProject       = errors.New("open project")
	ErrSaveFunction0x                   = errors.New("save function 0x")

	ErrSetNoteNilArtifact = errors.New("set note: nil artifact")
	ErrStoreArtifact      = errors.New("store artifact")
	ErrStoreFunction      = errors.New("store function")
	ErrStoreInstruction   = errors.New("store instruction")
	ErrStoreReference     = errors.New("store reference")
	ErrStoreSection       = errors.New("store section")
)

// ErrProject is a sqlite file that is not a decomp program database.
var ErrProject = errs.Wrap(errs.ErrInvalid, "not a decomp project")

// Open opens a sqlite program database, creating it when the file is new.
// The saved lift is loaded into this process. A later Add, rename, comment,
// or accepted check is written back to the same file. Migrations come from
// the embedded sqlc schema through [schema.DBArg].
func Open(ctx context.Context, path string) (*Project, error) {
	if ctx == nil {
		return nil, errs.ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, ErrOpenProjectEmptyPath
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenOpenProject, err)
	}
	var arg schema.DBArg
	if err := arg.Parse(sqliteFileURL(absolute)); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenOpenProject, err)
	}
	return openConn(ctx, arg.Value(), absolute)
}

// OpenConn migrates and loads a program database on a parsed [db.Arg] connection.
func OpenConn(ctx context.Context, conn *db.Conn[schema.Queries]) (*Project, error) {
	if ctx == nil {
		return nil, errs.ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if conn == nil || strings.TrimSpace(conn.URL()) == "" {
		return nil, ErrOpenProjectEmptyPath
	}
	absolute, err := filepath.Abs(sqlitePath(conn.URL()))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenConnOpenProject, err)
	}
	return openConn(ctx, conn, absolute)
}

func openConn(ctx context.Context, conn *db.Conn[schema.Queries], absolute string) (*Project, error) {
	if err := rejectForeignDB(ctx, absolute); err != nil {
		return nil, err
	}
	if err := conn.Open(ctx, schema.FS, schema.New(conn.URL())); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpenConnOpenProject, err)
	}
	project := &Project{conn: conn, path: absolute}
	if err := project.load(ctx); err != nil {
		_ = project.Close()
		return nil, err
	}
	return project, nil
}

// Close releases the sqlite file. A memory project has nothing to close.
func (project *Project) Close() error {
	if project == nil || project.conn == nil {
		return nil
	}
	err := project.conn.Close()
	project.conn = nil
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCloseProject, err)
	}
	return nil
}

func (project *Project) queries() schema.Queries {
	if project == nil || project.conn == nil {
		return nil
	}
	return project.conn.Queries()
}

// Path is the sqlite file passed to [Open]. A memory project returns an empty path.
func (project *Project) Path() string {
	if project == nil {
		return ""
	}
	return project.path
}

// Artifact returns the open image with this name.
func (project *Project) Artifact(name string) (*Artifact, bool) {
	if project == nil {
		return nil, false
	}
	for _, artifact := range project.artifacts {
		if artifact.name == name {
			return artifact, true
		}
	}
	return nil, false
}

func sqlitePath(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Scheme == "file" && parsed.Host == "" && parsed.Opaque == "" && parsed.Path == raw {
		return raw
	}
	switch parsed.Scheme {
	case "file":
		if parsed.Path != "" {
			return parsed.Path
		}
	case "sqlite", "sqlite3":
		if parsed.Host != "" && parsed.Path != "" {
			return parsed.Host + parsed.Path
		}
		if parsed.Opaque != "" {
			return parsed.Opaque
		}
		if parsed.Path != "" {
			return parsed.Path
		}
	}
	return raw
}

func sqliteFileURL(path string) string {
	location := url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(path),
		RawQuery: "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
	}
	return location.String()
}

func rejectForeignDB(ctx context.Context, path string) error {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrRejectForeignDBOpenProject, err)
	}
	defer database.Close()
	var tables, artifact, migrated int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'`).Scan(&tables); err != nil {
		return fmt.Errorf("%w: %w", ErrRejectForeignDBOpenProject, err)
	}
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'artifact'`).Scan(&artifact); err != nil {
		return fmt.Errorf("%w: %w", ErrRejectForeignDBOpenProject, err)
	}
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&migrated); err != nil {
		return fmt.Errorf("%w: %w", ErrRejectForeignDBOpenProject, err)
	}
	if tables > 0 && artifact == 0 && migrated == 0 {
		return fmt.Errorf("%w: %s", ErrProject, path)
	}
	return nil
}

func (project *Project) load(ctx context.Context) error {
	rows, err := project.queries().ListArtifacts(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProjectLoadLoadProject, err)
	}
	var artifacts []*Artifact
	for _, row := range rows {
		artifact := &Artifact{project: project, symbols: map[uint64]string{}, notes: map[uint64]string{}}
		artifact.name = row.Name
		artifact.format = row.Format
		artifact.architecture = disasm.Architecture(row.Architecture)
		artifact.mode = disasm.Mode(row.Mode)
		artifact.entry = uint64(row.Entry)
		artifact.path = row.Path
		artifact.digest = row.Digest
		artifacts = append(artifacts, artifact)
	}
	for _, artifact := range artifacts {
		if err := artifact.load(ctx); err != nil {
			return err
		}
	}
	project.artifacts = artifacts
	return nil
}

func (artifact *Artifact) load(ctx context.Context) error {
	if err := artifact.loadSections(ctx); err != nil {
		return err
	}
	if err := artifact.loadSymbols(ctx); err != nil {
		return err
	}
	if err := artifact.loadFunctions(ctx); err != nil {
		return err
	}
	if err := artifact.loadNotes(ctx); err != nil {
		return err
	}
	return artifact.loadImageStrings(ctx)
}

func (artifact *Artifact) loadImageStrings(ctx context.Context) error {
	rows, err := artifact.project.queries().ListImageStrings(ctx, artifact.name)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrArtifactLoadImageStringsLoadStrings, err)
	}
	for _, row := range rows {
		artifact.imageStrings = append(artifact.imageStrings, cstring{Address: uint64(row.Address), Text: row.Text})
	}
	return nil
}

func (artifact *Artifact) loadNotes(ctx context.Context) error {
	rows, err := artifact.project.queries().ListNotes(ctx, artifact.name)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrArtifactLoadNotesLoadNotes, err)
	}
	if artifact.notes == nil {
		artifact.notes = map[uint64]string{}
	}
	for _, row := range rows {
		if row.Text == "" {
			continue
		}
		artifact.notes[uint64(row.Address)] = row.Text
	}
	return nil
}

func (artifact *Artifact) loadSections(ctx context.Context) error {
	rows, err := artifact.project.queries().ListSections(ctx, artifact.name)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoadSections, err)
	}
	for _, row := range rows {
		artifact.sections = append(artifact.sections, disasm.Section{
			Name:    row.Name,
			Address: uint64(row.Address),
			Size:    uint64(row.Size),
			Exec:    row.Executable != 0,
			Write:   row.Writable != 0,
			Read:    row.Readable != 0,
		})
	}
	return nil
}

func (artifact *Artifact) loadSymbols(ctx context.Context) error {
	rows, err := artifact.project.queries().ListSymbols(ctx, artifact.name)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrArtifactLoadSymbolsLoadSymbols, err)
	}
	for _, row := range rows {
		artifact.symbols[uint64(row.Address)] = row.Name
	}
	return nil
}

func (artifact *Artifact) loadFunctions(ctx context.Context) error {
	rows, err := artifact.project.queries().ListFunctions(ctx, artifact.name)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoadFunctions, err)
	}
	for _, row := range rows {
		artifact.functions = append(artifact.functions, Function{
			Address:   uint64(row.Address),
			Name:      row.Name,
			Signature: row.Signature,
			Comment:   row.Comment,
			Source:    row.Source,
			Size:      uint64(row.Size),
		})
	}
	instructions, err := artifact.loadInstructions(ctx)
	if err != nil {
		return err
	}
	blocks, err := artifact.loadBlocks(ctx)
	if err != nil {
		return err
	}
	references, err := artifact.loadReferences(ctx)
	if err != nil {
		return err
	}
	texts, err := artifact.loadStrings(ctx)
	if err != nil {
		return err
	}
	for index := range artifact.functions {
		function := &artifact.functions[index]
		function.Instructions = instructions[function.Address]
		owned := blocks[function.Address]
		cursor := 0
		for blockIndex := range owned {
			count := owned[blockIndex].count
			if cursor+count > len(function.Instructions) {
				return fmt.Errorf("load function 0x%x: %w", function.Address, ErrBlockRunsPastItsInstructions)
			}
			owned[blockIndex].block.Instructions = append([]disasm.Instruction(nil), function.Instructions[cursor:cursor+count]...)
			cursor += count
			function.Blocks = append(function.Blocks, owned[blockIndex].block)
		}
		if cursor != len(function.Instructions) {
			return fmt.Errorf("load function 0x%x: %w", function.Address, ErrBlocksDoNotCoverItsInstructions)
		}
		function.References = references[function.Address]
		function.Strings = texts[function.Address]
	}
	indexArtifact(artifact)
	return nil
}

type storedBlock struct {
	block Block
	count int
}

func (artifact *Artifact) loadInstructions(ctx context.Context) (map[uint64][]disasm.Instruction, error) {
	rows, err := artifact.project.queries().ListInstructions(ctx, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLoadInstructions, err)
	}
	out := map[uint64][]disasm.Instruction{}
	for _, row := range rows {
		out[uint64(row.FunctionAddress)] = append(out[uint64(row.FunctionAddress)], disasm.Instruction{
			Address:  uint64(row.Address),
			Size:     uint16(row.Size),
			Bytes:    row.Bytes,
			Mnemonic: row.Mnemonic,
			Operands: row.Operands,
			ID:       uint32(row.InstructionID),
		})
	}
	return out, nil
}

func (artifact *Artifact) loadBlocks(ctx context.Context) (map[uint64][]storedBlock, error) {
	rows, err := artifact.project.queries().ListBlocks(ctx, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArtifactLoadBlocksLoadBlocks, err)
	}
	out := map[uint64][]storedBlock{}
	for _, row := range rows {
		out[uint64(row.FunctionAddress)] = append(out[uint64(row.FunctionAddress)], storedBlock{
			block: Block{
				Address: uint64(row.Address),
				Kind:    row.Kind,
				Target:  uint64(row.Target),
				Fall:    uint64(row.Fall),
			},
			count: int(row.InstructionCount),
		})
	}
	return out, nil
}

func (artifact *Artifact) loadReferences(ctx context.Context) (map[uint64][]Reference, error) {
	rows, err := artifact.project.queries().ListReferences(ctx, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLoadReferences, err)
	}
	out := map[uint64][]Reference{}
	for _, row := range rows {
		out[uint64(row.FunctionAddress)] = append(out[uint64(row.FunctionAddress)], Reference{
			From: uint64(row.FromAddress),
			To:   uint64(row.ToAddress),
			Kind: row.Kind,
		})
	}
	return out, nil
}

func (artifact *Artifact) loadStrings(ctx context.Context) (map[uint64][]string, error) {
	rows, err := artifact.project.queries().ListFunctionStrings(ctx, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArtifactLoadStringsLoadStrings, err)
	}
	out := map[uint64][]string{}
	for _, row := range rows {
		out[uint64(row.FunctionAddress)] = append(out[uint64(row.FunctionAddress)], row.Text)
	}
	return out, nil
}

func (project *Project) insertArtifact(ctx context.Context, artifact *Artifact, image []byte) error {
	if project.conn == nil {
		return nil
	}
	err := project.conn.Tx(ctx, func(q schema.Queries) error {
		if err := q.InsertArtifact(ctx, schema.InsertArtifactParams{
			Name: artifact.name, Format: artifact.format,
			Architecture: int64(artifact.architecture), Mode: int64(artifact.mode), Entry: int64(artifact.entry),
			Image: image, Path: artifact.path, Digest: artifact.digest,
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrStoreArtifact, err)
		}
		for position, section := range artifact.sections {
			executable, writable, readable := int64(0), int64(0), int64(0)
			if section.Exec {
				executable = 1
			}
			if section.Write {
				writable = 1
			}
			if section.Read {
				readable = 1
			}
			if err := q.InsertSection(ctx, schema.InsertSectionParams{
				Artifact: artifact.name, Position: int64(position), Name: section.Name,
				Address: int64(section.Address), Size: int64(section.Size),
				Executable: executable, Writable: writable, Readable: readable,
			}); err != nil {
				return fmt.Errorf("%w: %w", ErrStoreSection, err)
			}
		}
		for address, name := range artifact.symbols {
			if err := q.InsertSymbol(ctx, schema.InsertSymbolParams{Artifact: artifact.name, Address: int64(address), Name: name}); err != nil {
				return fmt.Errorf("%w: %w", ErrProjectInsertArtifactStoreSymbol, err)
			}
		}
		for _, function := range artifact.functions {
			if err := insertFunction(ctx, q, artifact.name, function); err != nil {
				return err
			}
		}
		for _, item := range artifact.imageStrings {
			if err := q.InsertImageString(ctx, schema.InsertImageStringParams{Artifact: artifact.name, Address: int64(item.Address), Text: item.Text}); err != nil {
				return fmt.Errorf("%w: %w", ErrProjectInsertArtifactStoreString, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStoreArtifact, err)
	}
	return nil
}

func insertFunction(ctx context.Context, q schema.Queries, artifact string, function Function) error {
	covered := 0
	for _, block := range function.Blocks {
		covered += len(block.Instructions)
	}
	if covered != len(function.Instructions) {
		return fmt.Errorf("store function 0x%x: %w", function.Address, ErrBlocksDoNotCoverItsInstructions)
	}
	if err := q.InsertFunction(ctx, schema.InsertFunctionParams{
		Artifact: artifact, Address: int64(function.Address), Name: function.Name,
		Signature: function.Signature, Comment: function.Comment, Source: function.Source, Size: int64(function.Size),
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrStoreFunction, err)
	}
	for position, instruction := range function.Instructions {
		raw := instruction.Bytes
		if raw == nil {
			raw = []byte{}
		}
		if err := q.InsertInstruction(ctx, schema.InsertInstructionParams{
			Artifact: artifact, FunctionAddress: int64(function.Address), Position: int64(position),
			Address: int64(instruction.Address), Size: int64(instruction.Size), Bytes: raw,
			Mnemonic: instruction.Mnemonic, Operands: instruction.Operands, InstructionID: int64(instruction.ID),
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrStoreInstruction, err)
		}
	}
	for position, block := range function.Blocks {
		if err := q.InsertBlock(ctx, schema.InsertBlockParams{
			Artifact: artifact, FunctionAddress: int64(function.Address), Position: int64(position),
			Address: int64(block.Address), Kind: block.Kind, Target: int64(block.Target), Fall: int64(block.Fall),
			InstructionCount: int64(len(block.Instructions)),
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrInsertFunctionStoreBlock, err)
		}
	}
	for position, reference := range function.References {
		if err := q.InsertReference(ctx, schema.InsertReferenceParams{
			Artifact: artifact, FunctionAddress: int64(function.Address), Position: int64(position),
			FromAddress: int64(reference.From), ToAddress: int64(reference.To), Kind: reference.Kind,
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrStoreReference, err)
		}
	}
	for position, text := range function.Strings {
		if err := q.InsertFunctionString(ctx, schema.InsertFunctionStringParams{
			Artifact: artifact, FunctionAddress: int64(function.Address), Position: int64(position), Text: text,
		}); err != nil {
			return fmt.Errorf("%w: %w", ErrInsertFunctionStoreString, err)
		}
	}
	return nil
}

func (artifact *Artifact) saveFunction(ctx context.Context, function *Function) error {
	if artifact == nil || artifact.project == nil || artifact.project.conn == nil {
		return nil
	}
	if ctx == nil {
		return errs.ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	updated, err := artifact.project.queries().UpdateFunction(ctx, schema.UpdateFunctionParams{
		Name: function.Name, Signature: function.Signature, Comment: function.Comment, Source: function.Source,
		Artifact: artifact.name, Address: int64(function.Address),
	})
	if err != nil {
		return fmt.Errorf("%w%x: %w", ErrSaveFunction0x, function.Address, err)
	}
	if updated != 1 {
		return fmt.Errorf("%w%x: stored %d rows", ErrSaveFunction0x, function.Address, updated)
	}
	return nil
}

func (artifact *Artifact) saveNote(ctx context.Context, address uint64, text string) error {
	if artifact == nil || artifact.project == nil || artifact.project.conn == nil {
		return nil
	}
	if ctx == nil {
		return errs.ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var err error
	if text == "" {
		err = artifact.project.queries().DeleteNote(ctx, schema.DeleteNoteParams{Artifact: artifact.name, Address: int64(address)})
	} else {
		err = artifact.project.queries().UpsertNote(ctx, schema.UpsertNoteParams{Artifact: artifact.name, Address: int64(address), Text: text})
	}
	if err != nil {
		return fmt.Errorf("%w%x: %w", ErrArtifactSaveNoteSaveNote0x, address, err)
	}
	return nil
}

// ArtifactFile returns the artifact stored for path.
// A missing file can still resolve to that stored image.
func (project *Project) ArtifactFile(path string) (*Artifact, bool) {
	if project == nil || path == "" || path == "-" {
		return nil, false
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, false
	}
	if artifact := project.byPath(absolute); artifact != nil {
		return artifact, true
	}
	artifact, ok := project.Artifact(filepath.Base(absolute))
	if !ok {
		return nil, false
	}
	if artifact.path == "" || artifact.path == absolute {
		return artifact, true
	}
	return nil, false
}

func (project *Project) adoptPath(ctx context.Context, artifact *Artifact, path string) error {
	artifact.path = path
	if project == nil || project.conn == nil {
		return nil
	}
	if err := project.queries().UpdateArtifactPath(ctx, schema.UpdateArtifactPathParams{Path: path, Name: artifact.name}); err != nil {
		return fmt.Errorf("%w: %w", ErrProjectAdoptPathStorePath, err)
	}
	return nil
}
