-- name: ListArtifacts :many
SELECT name, format, architecture, mode, entry, path, digest
FROM artifact
ORDER BY rowid;

-- name: InsertArtifact :exec
INSERT INTO artifact (name, format, architecture, mode, entry, image, path, digest)
VALUES (sqlc.arg(name), sqlc.arg(format), sqlc.arg(architecture), sqlc.arg(mode), sqlc.arg(entry), sqlc.arg(image), sqlc.arg(path), sqlc.arg(digest));

-- name: UpdateArtifactPath :exec
UPDATE artifact SET path = sqlc.arg(path) WHERE name = sqlc.arg(name);

-- name: ListSections :many
SELECT name, address, size, executable, writable, readable
FROM section
WHERE artifact = ?
ORDER BY position;

-- name: InsertSection :exec
INSERT INTO section (artifact, position, name, address, size, executable, writable, readable)
VALUES (sqlc.arg(artifact), sqlc.arg(position), sqlc.arg(name), sqlc.arg(address), sqlc.arg(size), sqlc.arg(executable), sqlc.arg(writable), sqlc.arg(readable));

-- name: ListSymbols :many
SELECT address, name FROM symbol WHERE artifact = ?;

-- name: InsertSymbol :exec
INSERT INTO symbol (artifact, address, name)
VALUES (sqlc.arg(artifact), sqlc.arg(address), sqlc.arg(name));

-- name: ListFunctions :many
SELECT address, name, signature, comment, source, size
FROM function
WHERE artifact = ?
ORDER BY address;

-- name: InsertFunction :exec
INSERT INTO function (artifact, address, name, signature, comment, source, size)
VALUES (sqlc.arg(artifact), sqlc.arg(address), sqlc.arg(name), sqlc.arg(signature), sqlc.arg(comment), sqlc.arg(source), sqlc.arg(size));

-- name: UpdateFunction :execrows
UPDATE function
SET name = sqlc.arg(name), signature = sqlc.arg(signature), comment = sqlc.arg(comment), source = sqlc.arg(source)
WHERE artifact = sqlc.arg(artifact) AND address = sqlc.arg(address);

-- name: ListInstructions :many
SELECT function_address, address, size, bytes, mnemonic, operands, instruction_id
FROM instruction
WHERE artifact = ?
ORDER BY function_address, position;

-- name: InsertInstruction :exec
INSERT INTO instruction (artifact, function_address, position, address, size, bytes, mnemonic, operands, instruction_id)
VALUES (sqlc.arg(artifact), sqlc.arg(function_address), sqlc.arg(position), sqlc.arg(address), sqlc.arg(size), sqlc.arg(bytes), sqlc.arg(mnemonic), sqlc.arg(operands), sqlc.arg(instruction_id));

-- name: ListBlocks :many
SELECT function_address, address, kind, target, fall, instruction_count
FROM block
WHERE artifact = ?
ORDER BY function_address, position;

-- name: InsertBlock :exec
INSERT INTO block (artifact, function_address, position, address, kind, target, fall, instruction_count)
VALUES (sqlc.arg(artifact), sqlc.arg(function_address), sqlc.arg(position), sqlc.arg(address), sqlc.arg(kind), sqlc.arg(target), sqlc.arg(fall), sqlc.arg(instruction_count));

-- name: ListReferences :many
SELECT function_address, from_address, to_address, kind
FROM reference
WHERE artifact = ?
ORDER BY function_address, position;

-- name: InsertReference :exec
INSERT INTO reference (artifact, function_address, position, from_address, to_address, kind)
VALUES (sqlc.arg(artifact), sqlc.arg(function_address), sqlc.arg(position), sqlc.arg(from_address), sqlc.arg(to_address), sqlc.arg(kind));

-- name: ListFunctionStrings :many
SELECT function_address, text
FROM function_string
WHERE artifact = ?
ORDER BY function_address, position;

-- name: InsertFunctionString :exec
INSERT INTO function_string (artifact, function_address, position, text)
VALUES (sqlc.arg(artifact), sqlc.arg(function_address), sqlc.arg(position), sqlc.arg(text));

-- name: ListNotes :many
SELECT address, text FROM note WHERE artifact = ? ORDER BY address;

-- name: UpsertNote :exec
INSERT INTO note (artifact, address, text)
VALUES (sqlc.arg(artifact), sqlc.arg(address), sqlc.arg(text))
ON CONFLICT (artifact, address) DO UPDATE SET text = excluded.text;

-- name: DeleteNote :exec
DELETE FROM note WHERE artifact = ? AND address = ?;

-- name: ListImageStrings :many
SELECT address, text FROM image_string WHERE artifact = ? ORDER BY address;

-- name: InsertImageString :exec
INSERT INTO image_string (artifact, address, text)
VALUES (sqlc.arg(artifact), sqlc.arg(address), sqlc.arg(text));
