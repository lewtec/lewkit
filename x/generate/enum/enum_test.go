package enum

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/generate"
	"github.com/stretchr/testify/require"
)

func TestRunScansPackage(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	decoy := filepath.Join(root, "sample")
	require.NoError(t, os.Mkdir(pkg, 0o755))
	require.NoError(t, os.Mkdir(decoy, 0o755))
	writeMarked(t, filepath.Join(pkg, "format.go"), "Format", "text")
	writeMarked(t, filepath.Join(pkg, "level.go"), "Level", "warning")
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "plain.go"), []byte("package sample\n\ntype Kind int\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "note.go"), []byte("package sample\n\n// lewkit:enum is the mark.\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "skip_test.go"), []byte("package sample\n\n//lewkit:enum\ntype Skip string\n"), 0o644))
	writeMarked(t, filepath.Join(decoy, "other.go"), "Other", "other")
	t.Chdir(pkg)
	t.Setenv("GOFILE", "format.go")
	t.Setenv("GOPACKAGE", "sample")
	require.NoError(t, Run(t.Context(), nil))
	require.FileExists(t, filepath.Join(pkg, "format_enum.go"))
	require.FileExists(t, filepath.Join(pkg, "level_enum.go"))
	require.NoFileExists(t, filepath.Join(pkg, "plain_enum.go"))
	require.NoFileExists(t, filepath.Join(pkg, "note_enum.go"))
	require.NoFileExists(t, filepath.Join(pkg, "skip_test_enum.go"))
	require.NoFileExists(t, filepath.Join(decoy, "other_enum.go"))
}

func TestRunUsesGOFILEDirectory(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	elsewhere := filepath.Join(root, "elsewhere")
	require.NoError(t, os.Mkdir(pkg, 0o755))
	require.NoError(t, os.Mkdir(elsewhere, 0o755))
	writeMarked(t, filepath.Join(pkg, "format.go"), "Format", "text")
	writeMarked(t, filepath.Join(pkg, "level.go"), "Level", "warning")
	writeMarked(t, filepath.Join(elsewhere, "other.go"), "Other", "other")
	t.Chdir(elsewhere)
	t.Setenv("GOFILE", filepath.Join(pkg, "format.go"))
	require.NoError(t, Run(t.Context(), nil))
	require.FileExists(t, filepath.Join(pkg, "format_enum.go"))
	require.FileExists(t, filepath.Join(pkg, "level_enum.go"))
	require.NoFileExists(t, filepath.Join(elsewhere, "other_enum.go"))
}

func TestRunUsesRelativeGOFILEDirectory(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	require.NoError(t, os.Mkdir(pkg, 0o755))
	writeMarked(t, filepath.Join(pkg, "format.go"), "Format", "text")
	writeMarked(t, filepath.Join(root, "other.go"), "Other", "other")
	t.Chdir(root)
	t.Setenv("GOFILE", filepath.Join("pkg", "format.go"))
	require.NoError(t, Run(t.Context(), nil))
	require.FileExists(t, filepath.Join(pkg, "format_enum.go"))
	require.NoFileExists(t, filepath.Join(root, "other_enum.go"))
}

func TestRunEmptyPackage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plain.go"), []byte("package plain\n"), 0o644))
	t.Chdir(dir)
	t.Setenv("GOFILE", "")
	err := Run(t.Context(), nil)
	require.ErrorIs(t, err, ErrNoEnum)
}

func TestRunSkipsDirectiveFile(t *testing.T) {
	dir := t.TempDir()
	writeMarked(t, filepath.Join(dir, "format.go"), "Format", "text")
	enumPath := filepath.Join(dir, "enum.go")
	directive := "package sample\n\n//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum\n"
	require.NoError(t, os.WriteFile(enumPath, []byte(directive), 0o644))
	t.Chdir(dir)
	t.Setenv("GOFILE", "enum.go")
	require.NoError(t, Run(t.Context(), nil))
	require.FileExists(t, filepath.Join(dir, "format_enum.go"))
	require.NoFileExists(t, filepath.Join(dir, "enum_enum.go"))
}

func TestRunRejectsEnumFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "enum.go")
	src := "package sample\n\n//lewkit:enum\ntype Kind string\n\nconst (\n\tKindA Kind = \"a\"\n)\n"
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	err := Run(t.Context(), []string{path})
	require.ErrorIs(t, err, ErrEnum)
	require.NoFileExists(t, filepath.Join(dir, "enum_enum.go"))
}

func TestRunExplicitFileSkipsSiblings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	format := filepath.Join(dir, "format.go")
	writeMarked(t, format, "Format", "text")
	writeMarked(t, filepath.Join(dir, "level.go"), "Level", "warning")
	require.NoError(t, Run(t.Context(), []string{format}))
	require.FileExists(t, filepath.Join(dir, "format_enum.go"))
	require.NoFileExists(t, filepath.Join(dir, "level_enum.go"))
}

func writeMarked(t *testing.T, path, name, token string) {
	t.Helper()
	src := fmt.Sprintf("package sample\n\n//lewkit:enum\ntype %s string\n\nconst (\n\t%sValue %s = %q\n)\n", name, name, name, token)
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
}

func TestRunRejectsUnmarkedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.go")
	require.NoError(t, os.WriteFile(path, []byte("package plain\n\ntype Kind int\n"), 0o644))
	err := Run(t.Context(), []string{path})
	require.ErrorIs(t, err, ErrNoEnum)
}

func TestRunRejectsBadMark(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.go")
	src := "package bad\n\n//lewkit:enum sideways\ntype Kind int\n"
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	err := Run(t.Context(), []string{path})
	require.ErrorIs(t, err, ErrEnum)
}

func TestRunCancelled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := Run(ctx, []string{"format.go"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestParseMembers(t *testing.T) {
	t.Parallel()
	src := `package sample

// Format is a rendering.
//
//lewkit:enum fold
type Format int

const (
	FormatText Format = iota + 1
	FormatTable
	FormatSARIF // sarif
)

// Level is a severity.
//
//lewkit:enum fold empty=warning
type Level string

const (
	LevelError Level = "error"
	LevelWarning Level = "warning" // warn, level/warn
	LevelNote Level = "note"
)

// Pool is a resource.
//
//lewkit:enum invalid=unknown
type Pool int

const (
	PoolControl Pool = iota // Unlimited orchestrator
	PoolIO
)

const Answer = "not a member"

// Vendor is an id.
//
//lewkit:enum
type Vendor uint32

const (
	VendorAMD Vendor = 0x1002
	VendorBit Vendor = 1 << 4
)

// Status is a reply.
//
//lewkit:enum
type Status string

const (
	StatusNoActivity Status = "no activity"
)
`
	parsed, err := parseSource("sample.go", []byte(src))
	require.NoError(t, err)
	require.Equal(t, "sample", parsed.pkg)
	require.Len(t, parsed.enums, 5)

	format := parsed.enums[0]
	require.Equal(t, "Format", format.name)
	require.True(t, format.fold)
	require.Equal(t, []member{
		{name: "FormatText", token: "text"},
		{name: "FormatTable", token: "table"},
		{name: "FormatSARIF", token: "sarif"},
	}, format.members)

	level := parsed.enums[1]
	require.True(t, level.stringKind)
	require.Equal(t, "warning", level.empty)
	require.Equal(t, "LevelWarning", level.members[1].name)
	require.Equal(t, []string{"warn", "level/warn"}, level.members[1].aliases)

	pool := parsed.enums[2]
	require.True(t, pool.hasInvalid)
	require.Equal(t, "unknown", pool.invalid)
	require.Equal(t, "control", pool.members[0].token)
	require.Empty(t, pool.members[0].aliases)

	require.Equal(t, "no activity", parsed.enums[4].members[0].token)
}

func TestRenderNotice(t *testing.T) {
	t.Parallel()
	src := `//go:build linux

package sample

//lewkit:enum
type Format int

const (
	FormatText Format = iota + 1
	FormatTable
)
`
	parsed, err := parseSource("format.go", []byte(src))
	require.NoError(t, err)
	buf, err := render(parsed)
	require.NoError(t, err)
	text := string(buf)
	require.Contains(t, text, "//go:build linux")
	require.Contains(t, text, generate.Notice("enum"))
	require.Contains(t, text, "func (f Format) String() string")
	require.Contains(t, text, `case FormatText:
		return "text"`)
	require.Contains(t, text, "func ParseFormat(text string) (Format, error)")
	require.Contains(t, text, "func (f *Format) Parse(text string) error")
	require.Contains(t, text, "func (Format) ArgChoices() []string")
	require.Contains(t, text, "func (f Format) MarshalText() ([]byte, error)")
	require.Contains(t, text, "func (f *Format) UnmarshalText(text []byte) error")
	require.Contains(t, text, "ErrFormat = errs.Wrap(errs.ErrUnknown, \"unknown format\")")
}

func TestRepeatedValue(t *testing.T) {
	t.Parallel()
	src := `package sample

//lewkit:enum
type Kind int

const (
	KindA Kind = 1
	KindB Kind = 1
)
`
	_, err := parseSource("kind.go", []byte(src))
	require.ErrorIs(t, err, ErrEnum)
}

func TestEmptyOptionMustNameMember(t *testing.T) {
	t.Parallel()
	src := `package sample

//lewkit:enum empty=missing
type Level string

const (
	LevelWarning Level = "warning"
)
`
	_, err := parseSource("level.go", []byte(src))
	require.ErrorIs(t, err, ErrEnum)
	require.ErrorIs(t, err, errs.ErrInvalid)
}

func TestGeneratedBehavior(t *testing.T) {
	dir := t.TempDir()
	src := `package sample

// Format is a rendering.
//
//lewkit:enum fold
type Format int

const (
	FormatText Format = iota + 1
	FormatTable
	FormatSARIF
)

// Level is a severity.
//
//lewkit:enum fold empty=warning
type Level string

const (
	LevelError Level = "error"
	LevelWarning Level = "warning"
	LevelNote Level = "note"
)

// Arch is a machine.
//
//lewkit:enum fold
type Arch int

const (
	ArchX86 Arch = 3 // x86, x86_64, amd64
)

// Pool is a resource.
//
//lewkit:enum invalid=unknown
type Pool int

const (
	PoolControl Pool = iota
	PoolIO
)

// Vendor is an id.
//
//lewkit:enum
type Vendor uint32

const (
	VendorAMD Vendor = 0x1002
	VendorBit Vendor = 1 << 4
)
`
	path := filepath.Join(dir, "format.go")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	require.NoError(t, Run(t.Context(), []string{path}))
	first, err := os.ReadFile(filepath.Join(dir, "format_enum.go"))
	require.NoError(t, err)
	require.NoError(t, Run(t.Context(), []string{path}))
	second, err := os.ReadFile(filepath.Join(dir, "format_enum.go"))
	require.NoError(t, err)
	require.True(t, bytes.Equal(first, second))

	mod := "module example.com/enumsample\n\ngo 1.27.0\n\nrequire github.com/lewtec/lewkit v0.0.0\n\nreplace github.com/lewtec/lewkit => " + moduleRoot(t) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "format_test.go"), []byte(behaviorTest), 0o644))

	cmd := exec.CommandContext(t.Context(), "go", "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOSUMDB=off")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent)
		dir = parent
	}
}

const behaviorTest = `package sample

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/errs"
)

func TestFormat(t *testing.T) {
	if int(FormatText) != 1 || FormatText.String() != "text" || FormatTable.String() != "table" {
		t.Fatalf("format text %d %s", FormatText, FormatText)
	}
	if !FormatSARIF.IsValid() || Format(0).IsValid() {
		t.Fatal("valid")
	}
	got, err := ParseFormat(" TABLE ")
	if err != nil || got != FormatTable {
		t.Fatalf("parse %v %v", got, err)
	}
	if _, err = ParseFormat("html"); !errors.Is(err, ErrFormat) || !errors.Is(err, errs.ErrUnknown) {
		t.Fatal(err)
	}
	raw, err := json.Marshal(FormatSARIF)
	if err != nil || string(raw) != "\"sarif\"" {
		t.Fatalf("json %s %v", raw, err)
	}
	var back Format
	if err = json.Unmarshal([]byte("\"text\""), &back); err != nil || back != FormatText {
		t.Fatal(err)
	}
	if err = json.Unmarshal([]byte("\"nope\""), &back); !errors.Is(err, ErrFormat) {
		t.Fatal(err)
	}
	choices := Format(0).ArgChoices()
	if len(choices) != 3 || choices[0] != "text" || choices[2] != "sarif" {
		t.Fatal(choices)
	}
	values := Format(0).Values()
	if len(values) != 3 || values[1] != FormatTable {
		t.Fatal(values)
	}

	type direct struct {
		Format Format ` + "`long:\"format\" help:\"rendering\"`" + `
	}
	text, err := cmd.Usage[direct]("tool")
	if err != nil || !strings.Contains(text, "choices: text, table, sarif") {
		t.Fatalf("usage %v %s", err, text)
	}
	parsed, err := cmd.Parse[direct]("--format", "SARIF")
	if err != nil || parsed.Format != FormatSARIF {
		t.Fatalf("flag %v %v", parsed.Format, err)
	}
	if _, err = cmd.Parse[direct]("--format", "html"); !errors.Is(err, ErrFormat) {
		t.Fatal(err)
	}

	type wrapped struct {
		Format cmd.EnumArg[Format] ` + "`long:\"format\"`" + `
	}
	flag, err := cmd.Parse[wrapped]("--format", "table")
	if err != nil || flag.Format.Value() != FormatTable {
		t.Fatal(err)
	}
}

func TestLevelArchPoolVendor(t *testing.T) {
	got, err := ParseLevel("")
	if err != nil || got != LevelWarning || got.String() != "warning" {
		t.Fatalf("empty %s %v", got, err)
	}
	got, err = ParseLevel("ERROR")
	if err != nil || got != LevelError {
		t.Fatal(err)
	}
	arch, err := ParseArch("AMD64")
	if err != nil || arch != ArchX86 || arch.String() != "x86" {
		t.Fatalf("arch %s %v", arch, err)
	}
	if _, err = ParseArch("arm64"); err == nil {
		t.Fatal("arm64")
	}
	if PoolControl.String() != "control" || Pool(9).String() != "unknown" || Pool(9).IsValid() {
		t.Fatalf("pool %s %s", PoolControl, Pool(9))
	}
	if uint32(VendorAMD) != 0x1002 || uint32(VendorBit) != 16 || !VendorBit.IsValid() {
		t.Fatal("vendor")
	}
}
`

func TestComposeEnum(t *testing.T) {
	t.Parallel()
	src := `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux" // gnu
)

//lewkit:enum
type Arch string

const (
	ArchAMD64 Arch = "amd64" // x86_64, x64
	ArchARM Arch = "arm"
)

//lewkit:enum compose=OS/Arch
type Platform string

const (
	PlatformLinuxAMD64 = Platform(string(OSLinux) + "/" + string(ArchAMD64)) // linux_amd64, linux-64
	PlatformLinuxARM = Platform((string(OSLinux) + "/" + string(ArchARM)))
)
`
	parsed, err := parseSource("platform.go", []byte(src))
	require.NoError(t, err)
	require.Len(t, parsed.enums, 3)
	platform := parsed.enums[2]
	require.Equal(t, "OS", platform.composeLeft)
	require.Equal(t, "Arch", platform.composeRight)
	require.Equal(t, "linux/amd64", platform.members[0].token)
	require.Equal(t, []string{
		"linux_amd64",
		"linux/x86_64",
		"linux_x86_64",
		"linux/x64",
		"linux_x64",
		"gnu/amd64",
		"gnu_amd64",
		"gnu/x86_64",
		"gnu_x86_64",
		"gnu/x64",
		"gnu_x64",
		"linux-64",
	}, platform.members[0].aliases)
	require.Equal(t, "linux/arm", platform.members[1].token)
	require.Equal(t, []string{"linux_arm", "gnu/arm", "gnu_arm"}, platform.members[1].aliases)

	buf, err := render(parsed)
	require.NoError(t, err)
	text := string(buf)
	require.Contains(t, text, "func (p Platform) OS() OS")
	require.Contains(t, text, "func (p Platform) Arch() Arch")
	require.Contains(t, text, "text, _, ok := strings.Cut(string(p), \"/\")")
	require.Contains(t, text, "_, text, ok := strings.Cut(string(p), \"/\")")
	require.Contains(t, text, "return OS(text)")
	require.Contains(t, text, "return Arch(text)")
	require.Contains(t, text, `"linux/amd64", "linux_amd64", "linux/x86_64"`)
}

func TestComposeRejects(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"literal": `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux"
)

//lewkit:enum
type Arch string

const (
	ArchAMD64 Arch = "amd64"
)

//lewkit:enum compose=OS/Arch
type Platform string

const (
	PlatformLinuxAMD64 Platform = "linux/amd64"
)
`,
		"missing": `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux"
)

//lewkit:enum compose=OS/Arch
type Platform string

const (
	PlatformLinuxAMD64 Platform = string(OSLinux) + "/" + string(OSLinux)
)
`,
		"integer": `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux"
)

//lewkit:enum
type Arch string

const (
	ArchAMD64 Arch = "amd64"
)

//lewkit:enum compose=OS/Arch
type Platform int

const (
	PlatformA Platform = 1
)
`,
		"separator": `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux/gnu"
)

//lewkit:enum
type Arch string

const (
	ArchAMD64 Arch = "amd64"
)

//lewkit:enum compose=OS/Arch
type Platform string

const (
	PlatformLinuxAMD64 = Platform(string(OSLinux) + "/" + string(ArchAMD64))
)
`,
		"bare": `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux"
)

//lewkit:enum
type Arch string

const (
	ArchAMD64 Arch = "amd64"
)

//lewkit:enum compose=OS/Arch
type Platform string

const (
	PlatformLinuxAMD64 Platform = string(OSLinux) + "/" + string(ArchAMD64)
)
`,
		"part": `package sample

//lewkit:enum
type OS string

const (
	OSLinux OS = "linux"
)

//lewkit:enum
type Arch string

const (
	ArchAMD64 Arch = "amd64"
)

const Also = "amd64"

//lewkit:enum compose=OS/Arch
type Platform string

const (
	PlatformLinuxAMD64 = Platform(string(OSLinux) + "/" + string(Also))
)
`,
		"option": `package sample

//lewkit:enum compose=OS
type Platform string

const (
	PlatformLinuxAMD64 Platform = "linux/amd64"
)
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseSource(name+".go", []byte(src))
			require.ErrorIs(t, err, ErrEnum)
		})
	}
}
