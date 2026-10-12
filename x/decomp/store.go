package decomp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/lewtec/lewkit/x/disasm"

	_ "modernc.org/sqlite"
)

// ErrProject is a sqlite file that is not a decomp program database.
var ErrProject = errors.New("not a decomp project")

const projectSchemaVersion = "2"

// Open opens a sqlite program database, creating it when the file is new.
// The saved lift is loaded into this process. A later Add, rename, comment,
// or accepted match is written back to the same file.
func Open(ctx context.Context, path string) (*Project, error) {
	if ctx == nil {
		return nil, errors.New("open project: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errors.New("open project: empty path")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	database, err := sql.Open("sqlite", sqliteDSN(absolute))
	if err != nil {
		return nil, fmt.Errorf("open project: %w", err)
	}
	database.SetMaxOpenConns(1)
	project := &Project{db: database, path: absolute}
	if err := project.prepare(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	if err := project.load(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	return project, nil
}

// Close releases the sqlite file. A memory project has nothing to close.
func (project *Project) Close() error {
	if project == nil || project.db == nil {
		return nil
	}
	err := project.db.Close()
	project.db = nil
	if err != nil {
		return fmt.Errorf("close project: %w", err)
	}
	return nil
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

func sqliteDSN(path string) string {
	location := url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(path),
		RawQuery: "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
	}
	return location.String()
}

func (project *Project) prepare(ctx context.Context) error {
	var tables int
	if err := project.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables); err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	var meta int
	if err := project.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'meta'`).Scan(&meta); err != nil {
		return fmt.Errorf("open project: %w", err)
	}
	if meta == 0 {
		if tables != 0 {
			return fmt.Errorf("%w: %s", ErrProject, project.path)
		}
		for _, statement := range projectSchema {
			if _, err := project.db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("create project: %w", err)
			}
		}
		return nil
	}
	var version string
	if err := project.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = 'schema'`).Scan(&version); err != nil {
		return fmt.Errorf("%w: %s", ErrProject, project.path)
	}
	if version == projectSchemaVersion {
		return nil
	}
	if version == "1" {
		return project.migrateNotes(ctx)
	}
	return fmt.Errorf("%w: schema %s", ErrProject, version)
}

func (project *Project) migrateNotes(ctx context.Context) error {
	transaction, err := project.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate project: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, noteTable); err != nil {
		return fmt.Errorf("migrate project: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `UPDATE meta SET value = ? WHERE key = 'schema'`, projectSchemaVersion); err != nil {
		return fmt.Errorf("migrate project: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("migrate project: %w", err)
	}
	return nil
}

func (project *Project) load(ctx context.Context) error {
	rows, err := project.db.QueryContext(ctx, `SELECT name, format, architecture, mode, entry FROM artifact ORDER BY rowid`)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	defer rows.Close()
	var artifacts []*Artifact
	for rows.Next() {
		artifact := &Artifact{project: project, symbols: map[uint64]string{}, notes: map[uint64]string{}}
		var architecture, mode, entry int64
		if err := rows.Scan(&artifact.name, &artifact.format, &architecture, &mode, &entry); err != nil {
			return fmt.Errorf("load project: %w", err)
		}
		artifact.architecture = disasm.Architecture(architecture)
		artifact.mode = disasm.Mode(mode)
		artifact.entry = uint64(entry)
		artifacts = append(artifacts, artifact)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	for _, artifact := range artifacts {
		if err := artifact.load(ctx, project.db); err != nil {
			return err
		}
	}
	project.artifacts = artifacts
	return nil
}

func (artifact *Artifact) load(ctx context.Context, database *sql.DB) error {
	if err := artifact.loadSections(ctx, database); err != nil {
		return err
	}
	if err := artifact.loadSymbols(ctx, database); err != nil {
		return err
	}
	if err := artifact.loadFunctions(ctx, database); err != nil {
		return err
	}
	return artifact.loadNotes(ctx, database)
}

func (artifact *Artifact) loadNotes(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `SELECT address, text FROM note WHERE artifact = ? ORDER BY address`, artifact.name)
	if err != nil {
		return fmt.Errorf("load notes: %w", err)
	}
	defer rows.Close()
	if artifact.notes == nil {
		artifact.notes = map[uint64]string{}
	}
	for rows.Next() {
		var address int64
		var text string
		if err := rows.Scan(&address, &text); err != nil {
			return fmt.Errorf("load notes: %w", err)
		}
		if text == "" {
			continue
		}
		artifact.notes[uint64(address)] = text
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load notes: %w", err)
	}
	return nil
}

func (artifact *Artifact) loadSections(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `SELECT name, address, size, executable, writable, readable FROM section WHERE artifact = ? ORDER BY position`, artifact.name)
	if err != nil {
		return fmt.Errorf("load sections: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var section disasm.Section
		var address, size int64
		var executable, writable, readable int
		if err := rows.Scan(&section.Name, &address, &size, &executable, &writable, &readable); err != nil {
			return fmt.Errorf("load sections: %w", err)
		}
		section.Address = uint64(address)
		section.Size = uint64(size)
		section.Exec = executable != 0
		section.Write = writable != 0
		section.Read = readable != 0
		artifact.sections = append(artifact.sections, section)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load sections: %w", err)
	}
	return nil
}

func (artifact *Artifact) loadSymbols(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `SELECT address, name FROM symbol WHERE artifact = ?`, artifact.name)
	if err != nil {
		return fmt.Errorf("load symbols: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var address int64
		var name string
		if err := rows.Scan(&address, &name); err != nil {
			return fmt.Errorf("load symbols: %w", err)
		}
		artifact.symbols[uint64(address)] = name
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load symbols: %w", err)
	}
	return nil
}

func (artifact *Artifact) loadFunctions(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, `SELECT address, name, signature, comment, source, size FROM function WHERE artifact = ? ORDER BY address`, artifact.name)
	if err != nil {
		return fmt.Errorf("load functions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var function Function
		var address, size int64
		if err := rows.Scan(&address, &function.Name, &function.Signature, &function.Comment, &function.Source, &size); err != nil {
			return fmt.Errorf("load functions: %w", err)
		}
		function.Address = uint64(address)
		function.Size = uint64(size)
		artifact.functions = append(artifact.functions, function)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load functions: %w", err)
	}
	instructions, err := artifact.loadInstructions(ctx, database)
	if err != nil {
		return err
	}
	blocks, err := artifact.loadBlocks(ctx, database)
	if err != nil {
		return err
	}
	references, err := artifact.loadReferences(ctx, database)
	if err != nil {
		return err
	}
	texts, err := artifact.loadStrings(ctx, database)
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
				return fmt.Errorf("load function 0x%x: block runs past its instructions", function.Address)
			}
			owned[blockIndex].block.Instructions = append([]disasm.Instruction(nil), function.Instructions[cursor:cursor+count]...)
			cursor += count
			function.Blocks = append(function.Blocks, owned[blockIndex].block)
		}
		if cursor != len(function.Instructions) {
			return fmt.Errorf("load function 0x%x: blocks do not cover its instructions", function.Address)
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

func (artifact *Artifact) loadInstructions(ctx context.Context, database *sql.DB) (map[uint64][]disasm.Instruction, error) {
	rows, err := database.QueryContext(ctx, `SELECT function_address, address, size, bytes, mnemonic, operands, instruction_id FROM instruction WHERE artifact = ? ORDER BY function_address, position`, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("load instructions: %w", err)
	}
	defer rows.Close()
	out := map[uint64][]disasm.Instruction{}
	for rows.Next() {
		var functionAddress, address, size, identifier int64
		var instruction disasm.Instruction
		if err := rows.Scan(&functionAddress, &address, &size, &instruction.Bytes, &instruction.Mnemonic, &instruction.Operands, &identifier); err != nil {
			return nil, fmt.Errorf("load instructions: %w", err)
		}
		instruction.Address = uint64(address)
		instruction.Size = uint16(size)
		instruction.ID = uint32(identifier)
		out[uint64(functionAddress)] = append(out[uint64(functionAddress)], instruction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load instructions: %w", err)
	}
	return out, nil
}

func (artifact *Artifact) loadBlocks(ctx context.Context, database *sql.DB) (map[uint64][]storedBlock, error) {
	rows, err := database.QueryContext(ctx, `SELECT function_address, address, kind, target, fall, instruction_count FROM block WHERE artifact = ? ORDER BY function_address, position`, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("load blocks: %w", err)
	}
	defer rows.Close()
	out := map[uint64][]storedBlock{}
	for rows.Next() {
		var functionAddress, address, target, fall int64
		var count int
		var block Block
		if err := rows.Scan(&functionAddress, &address, &block.Kind, &target, &fall, &count); err != nil {
			return nil, fmt.Errorf("load blocks: %w", err)
		}
		block.Address = uint64(address)
		block.Target = uint64(target)
		block.Fall = uint64(fall)
		key := uint64(functionAddress)
		out[key] = append(out[key], storedBlock{block: block, count: count})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load blocks: %w", err)
	}
	return out, nil
}

func (artifact *Artifact) loadReferences(ctx context.Context, database *sql.DB) (map[uint64][]Reference, error) {
	rows, err := database.QueryContext(ctx, `SELECT function_address, from_address, to_address, kind FROM reference WHERE artifact = ? ORDER BY function_address, position`, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("load references: %w", err)
	}
	defer rows.Close()
	out := map[uint64][]Reference{}
	for rows.Next() {
		var functionAddress, from, to int64
		var reference Reference
		if err := rows.Scan(&functionAddress, &from, &to, &reference.Kind); err != nil {
			return nil, fmt.Errorf("load references: %w", err)
		}
		reference.From = uint64(from)
		reference.To = uint64(to)
		out[uint64(functionAddress)] = append(out[uint64(functionAddress)], reference)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load references: %w", err)
	}
	return out, nil
}

func (artifact *Artifact) loadStrings(ctx context.Context, database *sql.DB) (map[uint64][]string, error) {
	rows, err := database.QueryContext(ctx, `SELECT function_address, text FROM function_string WHERE artifact = ? ORDER BY function_address, position`, artifact.name)
	if err != nil {
		return nil, fmt.Errorf("load strings: %w", err)
	}
	defer rows.Close()
	out := map[uint64][]string{}
	for rows.Next() {
		var functionAddress int64
		var text string
		if err := rows.Scan(&functionAddress, &text); err != nil {
			return nil, fmt.Errorf("load strings: %w", err)
		}
		out[uint64(functionAddress)] = append(out[uint64(functionAddress)], text)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load strings: %w", err)
	}
	return out, nil
}

func (project *Project) insertArtifact(ctx context.Context, artifact *Artifact, image []byte) error {
	if project.db == nil {
		return nil
	}
	tx, err := project.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store artifact: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO artifact (name, format, architecture, mode, entry, image) VALUES (?, ?, ?, ?, ?, ?)`,
		artifact.name, artifact.format, int64(artifact.architecture), int64(artifact.mode), int64(artifact.entry), image); err != nil {
		return fmt.Errorf("store artifact: %w", err)
	}
	for position, section := range artifact.sections {
		executable, writable, readable := 0, 0, 0
		if section.Exec {
			executable = 1
		}
		if section.Write {
			writable = 1
		}
		if section.Read {
			readable = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO section (artifact, position, name, address, size, executable, writable, readable) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			artifact.name, position, section.Name, int64(section.Address), int64(section.Size), executable, writable, readable); err != nil {
			return fmt.Errorf("store section: %w", err)
		}
	}
	for address, name := range artifact.symbols {
		if _, err := tx.ExecContext(ctx, `INSERT INTO symbol (artifact, address, name) VALUES (?, ?, ?)`,
			artifact.name, int64(address), name); err != nil {
			return fmt.Errorf("store symbol: %w", err)
		}
	}
	for _, function := range artifact.functions {
		if err := insertFunction(ctx, tx, artifact.name, function); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store artifact: %w", err)
	}
	return nil
}

func insertFunction(ctx context.Context, tx *sql.Tx, artifact string, function Function) error {
	covered := 0
	for _, block := range function.Blocks {
		covered += len(block.Instructions)
	}
	if covered != len(function.Instructions) {
		return fmt.Errorf("store function 0x%x: blocks do not cover its instructions", function.Address)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO function (artifact, address, name, signature, comment, source, size) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		artifact, int64(function.Address), function.Name, function.Signature, function.Comment, function.Source, int64(function.Size)); err != nil {
		return fmt.Errorf("store function: %w", err)
	}
	for position, instruction := range function.Instructions {
		raw := instruction.Bytes
		if raw == nil {
			raw = []byte{}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO instruction (artifact, function_address, position, address, size, bytes, mnemonic, operands, instruction_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			artifact, int64(function.Address), position, int64(instruction.Address), int64(instruction.Size), raw, instruction.Mnemonic, instruction.Operands, int64(instruction.ID)); err != nil {
			return fmt.Errorf("store instruction: %w", err)
		}
	}
	for position, block := range function.Blocks {
		if _, err := tx.ExecContext(ctx, `INSERT INTO block (artifact, function_address, position, address, kind, target, fall, instruction_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			artifact, int64(function.Address), position, int64(block.Address), block.Kind, int64(block.Target), int64(block.Fall), len(block.Instructions)); err != nil {
			return fmt.Errorf("store block: %w", err)
		}
	}
	for position, reference := range function.References {
		if _, err := tx.ExecContext(ctx, `INSERT INTO reference (artifact, function_address, position, from_address, to_address, kind) VALUES (?, ?, ?, ?, ?, ?)`,
			artifact, int64(function.Address), position, int64(reference.From), int64(reference.To), reference.Kind); err != nil {
			return fmt.Errorf("store reference: %w", err)
		}
	}
	for position, text := range function.Strings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO function_string (artifact, function_address, position, text) VALUES (?, ?, ?, ?)`,
			artifact, int64(function.Address), position, text); err != nil {
			return fmt.Errorf("store string: %w", err)
		}
	}
	return nil
}

func (artifact *Artifact) saveFunction(ctx context.Context, function *Function) error {
	if artifact == nil || artifact.project == nil || artifact.project.db == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("save function: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	result, err := artifact.project.db.ExecContext(ctx, `UPDATE function SET name = ?, signature = ?, comment = ?, source = ? WHERE artifact = ? AND address = ?`,
		function.Name, function.Signature, function.Comment, function.Source, artifact.name, int64(function.Address))
	if err != nil {
		return fmt.Errorf("save function 0x%x: %w", function.Address, err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("save function 0x%x: %w", function.Address, err)
	}
	if updated != 1 {
		return fmt.Errorf("save function 0x%x: stored %d rows", function.Address, updated)
	}
	return nil
}

func (artifact *Artifact) saveNote(ctx context.Context, address uint64, text string) error {
	if artifact == nil || artifact.project == nil || artifact.project.db == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("save note: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var err error
	if text == "" {
		_, err = artifact.project.db.ExecContext(ctx, `DELETE FROM note WHERE artifact = ? AND address = ?`, artifact.name, int64(address))
	} else {
		_, err = artifact.project.db.ExecContext(ctx, `INSERT INTO note (artifact, address, text) VALUES (?, ?, ?)
			ON CONFLICT(artifact, address) DO UPDATE SET text = excluded.text`, artifact.name, int64(address), text)
	}
	if err != nil {
		return fmt.Errorf("save note 0x%x: %w", address, err)
	}
	return nil
}

const noteTable = `CREATE TABLE IF NOT EXISTS note (
	artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
	address INTEGER NOT NULL,
	text TEXT NOT NULL,
	PRIMARY KEY (artifact, address)
)`

var projectSchema = []string{
	`CREATE TABLE meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	`INSERT INTO meta (key, value) VALUES ('schema', '2')`,
	`CREATE TABLE artifact (
		name TEXT PRIMARY KEY,
		format TEXT NOT NULL,
		architecture INTEGER NOT NULL,
		mode INTEGER NOT NULL,
		entry INTEGER NOT NULL,
		image BLOB NOT NULL
	)`,
	`CREATE TABLE section (
		artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
		position INTEGER NOT NULL,
		name TEXT NOT NULL,
		address INTEGER NOT NULL,
		size INTEGER NOT NULL,
		executable INTEGER NOT NULL,
		writable INTEGER NOT NULL,
		readable INTEGER NOT NULL,
		PRIMARY KEY (artifact, position)
	)`,
	`CREATE TABLE symbol (
		artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
		address INTEGER NOT NULL,
		name TEXT NOT NULL,
		PRIMARY KEY (artifact, address)
	)`,
	`CREATE TABLE function (
		artifact TEXT NOT NULL REFERENCES artifact(name) ON DELETE CASCADE,
		address INTEGER NOT NULL,
		name TEXT NOT NULL,
		signature TEXT NOT NULL,
		comment TEXT NOT NULL,
		source TEXT NOT NULL,
		size INTEGER NOT NULL,
		PRIMARY KEY (artifact, address)
	)`,
	`CREATE TABLE instruction (
		artifact TEXT NOT NULL,
		function_address INTEGER NOT NULL,
		position INTEGER NOT NULL,
		address INTEGER NOT NULL,
		size INTEGER NOT NULL,
		bytes BLOB NOT NULL,
		mnemonic TEXT NOT NULL,
		operands TEXT NOT NULL,
		instruction_id INTEGER NOT NULL,
		PRIMARY KEY (artifact, function_address, position),
		FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
	)`,
	`CREATE TABLE block (
		artifact TEXT NOT NULL,
		function_address INTEGER NOT NULL,
		position INTEGER NOT NULL,
		address INTEGER NOT NULL,
		kind TEXT NOT NULL,
		target INTEGER NOT NULL,
		fall INTEGER NOT NULL,
		instruction_count INTEGER NOT NULL,
		PRIMARY KEY (artifact, function_address, position),
		FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
	)`,
	`CREATE TABLE reference (
		artifact TEXT NOT NULL,
		function_address INTEGER NOT NULL,
		position INTEGER NOT NULL,
		from_address INTEGER NOT NULL,
		to_address INTEGER NOT NULL,
		kind TEXT NOT NULL,
		PRIMARY KEY (artifact, function_address, position),
		FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
	)`,
	`CREATE TABLE function_string (
		artifact TEXT NOT NULL,
		function_address INTEGER NOT NULL,
		position INTEGER NOT NULL,
		text TEXT NOT NULL,
		PRIMARY KEY (artifact, function_address, position),
		FOREIGN KEY (artifact, function_address) REFERENCES function(artifact, address) ON DELETE CASCADE
	)`,
	noteTable,
}
