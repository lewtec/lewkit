package enum

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/generate"
	"github.com/stretchr/testify/require"
)

func TestRunRequiresFile(t *testing.T) {
	t.Parallel()
	err := Run(t.Context(), nil)
	require.ErrorIs(t, err, ErrFileRequired)
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
	LevelWarning Level = "warning" // warn
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
	require.False(t, format.emitConst)
	requireMembers(t, format.members, []member{
		{name: "FormatText", token: "text"},
		{name: "FormatTable", token: "table"},
		{name: "FormatSARIF", token: "sarif"},
	})

	level := parsed.enums[1]
	require.True(t, level.stringKind)
	require.Equal(t, "warning", level.empty)
	require.Equal(t, "LevelWarning", level.members[1].name)
	require.Equal(t, []string{"warn"}, level.members[1].aliases)

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

func requireMembers(t *testing.T, got, want []member) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range want {
		require.Equal(t, want[i].name, got[i].name)
		require.Equal(t, want[i].token, got[i].token)
		require.Equal(t, want[i].aliases, got[i].aliases)
	}
}

func TestDeclaredValues(t *testing.T) {
	t.Parallel()
	src := `package sample

//lewkit:enum empty=linux values=linux, darwin, windows fold
type OS string

//lewkit:enum invalid=unknown values=text,table
type Kind int

//lewkit:enum values=red,green
type Shade string

const (
	ShadeRed Shade = "red"
	ShadeGreen Shade = "green"
)

//lewkit:enum values=sim,build
type target int
`
	parsed, err := parseSource("os.go", []byte(src))
	require.NoError(t, err)
	require.Len(t, parsed.enums, 4)

	osEnum := parsed.enums[0]
	require.True(t, osEnum.emitConst)
	require.True(t, osEnum.fold)
	require.Equal(t, "linux", osEnum.empty)
	requireMembers(t, osEnum.members, []member{
		{name: "OSLinux", token: "linux"},
		{name: "OSDarwin", token: "darwin"},
		{name: "OSWindows", token: "windows"},
	})

	kind := parsed.enums[1]
	require.True(t, kind.emitConst)
	require.Equal(t, "unknown", kind.invalid)
	requireMembers(t, kind.members, []member{
		{name: "KindText", token: "text"},
		{name: "KindTable", token: "table"},
	})

	shade := parsed.enums[2]
	require.False(t, shade.emitConst)
	requireMembers(t, shade.members, []member{
		{name: "ShadeRed", token: "red"},
		{name: "ShadeGreen", token: "green"},
	})

	step := parsed.enums[3]
	require.True(t, step.emitConst)
	requireMembers(t, step.members, []member{
		{name: "targetSim", token: "sim"},
		{name: "targetBuild", token: "build"},
	})

	buf, err := render(parsed)
	require.NoError(t, err)
	text := string(buf)
	require.Contains(t, text, "OSLinux")
	require.Contains(t, text, "OS = \"linux\"")
	require.Contains(t, text, "KindText Kind = iota")
	require.Contains(t, text, "\tKindTable\n")
	require.NotContains(t, text, "ShadeRed Shade")
	require.Contains(t, text, "targetSim target = iota")
}

func TestDeclaredValuesMustMatch(t *testing.T) {
	t.Parallel()
	cases := []string{
		`package sample

//lewkit:enum values=red,green
type Shade string

const (
	ShadeRed Shade = "red"
	ShadeBlue Shade = "blue"
)
`,
		`package sample

//lewkit:enum values=text,table
type Kind int

const (
	KindText Kind = iota + 1
	KindTable
)
`,
		`package sample

//lewkit:enum values=red,green
type Shade string

const (
	ShadeRed Shade = "red" // crimson
	ShadeGreen Shade = "green"
)
`,
		`package sample

//lewkit:enum values=6502
type CPU int
`,
		`package sample

//lewkit:enum values=
type OS string
`,
		`package sample

//lewkit:enum values=linux values=darwin
type OS string
`,
	}
	for i, src := range cases {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			t.Parallel()
			_, err := parseSource("bad.go", []byte(src))
			require.ErrorIs(t, err, ErrEnum)
		})
	}
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

//lewkit:enum fold empty=linux values=linux,darwin,windows
type OS string

//lewkit:enum invalid=unknown values=text,table
type Kind int

//lewkit:enum values=red,green
type Shade string

const (
	ShadeRed Shade = "red"
	ShadeGreen Shade = "green"
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
	if OSLinux != "linux" || OSDarwin != "darwin" || OSWindows != "windows" {
		t.Fatal(OSLinux)
	}
	osName, err := ParseOS("")
	if err != nil || osName != OSLinux {
		t.Fatal(err)
	}
	osName, err = ParseOS("DARWIN")
	if err != nil || osName != OSDarwin {
		t.Fatal(err)
	}
	if KindText != 0 || KindTable != 1 || Kind(9).String() != "unknown" || Kind(9).IsValid() {
		t.Fatal(KindText)
	}
	if ShadeRed != "red" || ShadeGreen != "green" || !ShadeGreen.IsValid() {
		t.Fatal(ShadeRed)
	}
}
`
