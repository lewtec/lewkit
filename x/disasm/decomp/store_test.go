package decomp

import (
	"database/sql"
	"debug/elf"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/disasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectFileKeepsTheLift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "program.decomp")
	image := elfTextAndRodata([]byte{0x48, 0x8d, 0x3d, 0xf9, 0x0f, 0x00, 0x00, 0xc3}, append([]byte("hello"), 0))
	branch := disasm.ELF64(elf.EM_X86_64, []byte{0x74, 0x00, 0xc3})

	project, err := Open(t.Context(), path)
	require.NoError(t, err)
	first, err := project.AddBytes(t.Context(), "strings.elf", image)
	require.NoError(t, err)
	second, err := project.AddBytes(t.Context(), "branch.elf", branch)
	require.NoError(t, err)
	require.NoError(t, first.SetName(t.Context(), 0x1000, "greet"))
	require.NoError(t, first.SetSignature(t.Context(), 0x1000, "void greet(void)"))
	require.NoError(t, first.SetComment(t.Context(), 0x1000, "loads hello"))
	function, ok := second.Function(0x1000)
	require.True(t, ok)
	result, err := second.Check(t.Context(), 0x1000, echoCandidate(function, "taken", "void taken(void)"))
	require.NoError(t, err)
	require.True(t, result.Matched)
	require.NoError(t, project.Close())

	reopened, err := Open(t.Context(), path)
	require.NoError(t, err)
	defer reopened.Close()
	_, err = reopened.AddBytes(t.Context(), "strings.elf", image)
	assert.ErrorIs(t, err, ErrArtifact)

	stored, ok := reopened.Artifact("strings.elf")
	require.True(t, ok)
	assert.Equal(t, "elf", stored.Format())
	assert.Contains(t, stored.Summary(), "functions 1")
	greet, ok := stored.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "greet", greet.Name)
	assert.Equal(t, "void greet(void)", greet.Signature)
	assert.Equal(t, "loads hello", greet.Comment)
	assert.Contains(t, greet.Strings, "hello")
	listing, err := stored.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, "lea")
	assert.Contains(t, listing, "greet")

	branched, ok := reopened.Artifact("branch.elf")
	require.True(t, ok)
	taken, ok := branched.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "taken", taken.Name)
	assert.Contains(t, taken.Source, "void taken")
	require.GreaterOrEqual(t, len(taken.Blocks), 2)
	assert.Equal(t, "branch", taken.Blocks[0].Kind)
	assert.Equal(t, uint64(0x1002), taken.Blocks[0].Target)
	assert.Equal(t, uint64(0x1002), taken.Blocks[0].Fall)
	assert.Equal(t, "je", taken.Blocks[0].Instructions[0].Mnemonic)
}

func TestOlderProjectKeepsNamesAndNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "program.decomp")
	project, err := Open(t.Context(), path)
	require.NoError(t, err)
	require.NoError(t, project.Close())

	database, err := sql.Open("sqlite", sqliteDSN(path))
	require.NoError(t, err)
	_, err = database.Exec(`UPDATE meta SET value = '1' WHERE key = 'schema'`)
	require.NoError(t, err)
	_, err = database.Exec(`DROP TABLE note`)
	require.NoError(t, err)
	require.NoError(t, database.Close())

	project, err = Open(t.Context(), path)
	require.NoError(t, err)
	artifact, err := project.AddBytes(t.Context(), "branch.elf", disasm.ELF64(elf.EM_X86_64, []byte{0x90, 0xc3}))
	require.NoError(t, err)
	require.NoError(t, artifact.SetName(t.Context(), 0x1000, "greet"))
	require.NoError(t, artifact.SetNote(t.Context(), 0x1000, "entry"))
	require.NoError(t, artifact.SetNote(t.Context(), 0x1001, "padding"))
	require.NoError(t, project.Close())

	project, err = Open(t.Context(), path)
	require.NoError(t, err)
	artifact, ok := project.Artifact("branch.elf")
	require.True(t, ok)
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "greet", function.Name)
	text, ok := artifact.Note(0x1000)
	require.True(t, ok)
	assert.Equal(t, "entry", text)
	text, ok = artifact.Note(0x1001)
	require.True(t, ok)
	assert.Equal(t, "padding", text)
	listing, err := artifact.Listing(0x1000)
	require.NoError(t, err)
	assert.Contains(t, listing, "greet")
	assert.Contains(t, listing, "; note entry")
	assert.Contains(t, listing, "; note padding")
	require.NoError(t, artifact.SetNote(t.Context(), 0x1001, ""))
	_, ok = artifact.Note(0x1001)
	assert.False(t, ok)
	require.NoError(t, project.Close())

	project, err = Open(t.Context(), path)
	require.NoError(t, err)
	defer project.Close()
	artifact, ok = project.Artifact("branch.elf")
	require.True(t, ok)
	_, ok = artifact.Note(0x1001)
	assert.False(t, ok)
	text, ok = artifact.Note(0x1000)
	require.True(t, ok)
	assert.Equal(t, "entry", text)
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	_, err := Open(t.Context(), "")
	require.Error(t, err)
}

func TestOpenRejectsOtherDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "other.db")
	database, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = database.Exec(`CREATE TABLE other (id INTEGER)`)
	require.NoError(t, err)
	require.NoError(t, database.Close())
	_, err = Open(t.Context(), path)
	assert.ErrorIs(t, err, ErrProject)
}
