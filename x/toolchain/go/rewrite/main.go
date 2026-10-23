// Command rewrite turns the modot-placed Go 1.27 trees under
// x/toolchain/go/_upstream into packages this module can import.
//
// cmd/compile/internal and cmd/link/internal have different parents, so one
// package cannot import both. The placed trees keep those paths. This program
// drops the internal element (compile/internal/gc becomes src/compile/gc) and
// retargets std internal imports onto the placed copy under src/std. cmd/asm
// lands under src/asm the same way. It also
// turns the linker's and compiler's process exits into panics so one process
// can compile and then link. //go:cgo_ldflag is rejected outside the standard
// library, so those lines are dropped. modot replaces the placed trees on
// purpose.
// patch.json then adds SetDiag, MainArgs, Run, and ParseFS onto that text,
// after import paths have been rewritten, and does not contain those paths.
//
// Run it after `go tool modot codebase apply`. go generate at the repo root
// does both.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrCompileBaseExitWasNotPatched    = errors.New("compile base.Exit was not patched")
	ErrCompileErrorexitWasNotPatched   = errors.New("compile ErrorExit was not patched")
	ErrCompileHandlepanicWasNotPatched = errors.New("compile handlePanic was not patched")
	ErrCopyPkg                         = errors.New("")
	ErrLinkExitWasNotPatched           = errors.New("link Exit was not patched")
	ErrPatchLibAnchors                 = errors.New("anchors")
	ErrPatchLibEmptyAnchor             = errs.Wrap(errs.ErrEmpty, "empty anchor")
	ErrPatchLibPatch                   = errors.New("patch")
)

const moduleRoot = "github.com/lewtec/lewkit/x/toolchain/go/src/"

func main() {
	root := filepath.Join("x", "toolchain", "go")
	if _, err := os.Stat(filepath.Join(root, "_upstream", "compile")); err != nil {
		fmt.Fprintf(os.Stderr, "rewrite: %s is missing; run go tool modot codebase apply\n", filepath.Join(root, "_upstream"))
		os.Exit(1)
	}
	dst := filepath.Join(root, "src")
	if err := os.RemoveAll(dst); err != nil {
		fatal(err)
	}
	for _, pair := range packages {
		if err := copyPkg(filepath.Join(root, "_upstream", pair[0]), filepath.Join(dst, pair[1])); err != nil {
			fatal(fmt.Errorf("%s: %w", pair[0], err))
		}
	}
	if err := patchAll(dst); err != nil {
		fatal(err)
	}
	if err := patchLib(dst); err != nil {
		fatal(err)
	}
	if err := writeGenerated(dst); err != nil {
		fatal(err)
	}
	if err := patchAsmFlags(dst); err != nil {
		fatal(err)
	}
}

// patchAsmFlags keeps the assembler flags off flag.CommandLine.
// The driver writes the pointers. The lewkit process and the test binary
// keep their own flag set.
func patchAsmFlags(dst string) error {
	path := filepath.Join(dst, "asm", "flags", "flags.go")
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	old := `var (
	Debug      = flag.Bool("debug", false, "dump instructions as they are parsed")
	OutputFile = flag.String("o", "", "output file; default foo.o for /a/b/c/foo.s as first argument")
	TrimPath   = flag.String("trimpath", "", "remove prefix from recorded source file paths")
	Shared     = flag.Bool("shared", false, "generate code that can be linked into a shared library")
	Dynlink    = flag.Bool("dynlink", false, "support references to Go symbols defined in other shared libraries")
	Linkshared = flag.Bool("linkshared", false, "generate code that will be linked against Go shared libraries")
	AllErrors  = flag.Bool("e", false, "no limit on number of errors reported")
	SymABIs    = flag.Bool("gensymabis", false, "write symbol ABI information to output file, don't assemble")
	Importpath = flag.String("p", obj.UnlinkablePkg, "set expected package import to path")
	Spectre    = flag.String("spectre", "", "enable spectre mitigations in ` + "`list`" + ` (all, ret)")
	Std        = flag.Bool("std", false, "building standard library")
)`
	newVars := `var (
	Debug      *bool
	OutputFile *string
	TrimPath   *string
	Shared     *bool
	Dynlink    *bool
	Linkshared *bool
	AllErrors  *bool
	SymABIs    *bool
	Importpath *string
	Spectre    *string
	Std        *bool
)`
	next := strings.Replace(string(body), old, newVars, 1)
	if next == string(body) {
		return errors.New("asm flags var block was not patched")
	}
	oldInit := `func init() {
	flag.Var(&D, "D", "predefined symbol with optional simple value -D=identifier=value; can be set multiple times")
	flag.Var(&I, "I", "include directory; can be set multiple times")
	flag.BoolVar(&DebugV, "v", false, "print debug output")
	flag.Var(objabi.NewDebugFlag(&DebugFlags, nil), "d", "enable debugging settings; try -d help")
	objabi.AddVersionFlag() // -V
	objabi.Flagcount("S", "print assembly and machine code", &PrintOut)

	DebugFlags.CompressInstructions = 1
}`
	newInit := `func init() {
	fs := flag.NewFlagSet("asm", flag.ContinueOnError)
	Debug = fs.Bool("debug", false, "dump instructions as they are parsed")
	OutputFile = fs.String("o", "", "output file; default foo.o for /a/b/c/foo.s as first argument")
	TrimPath = fs.String("trimpath", "", "remove prefix from recorded source file paths")
	Shared = fs.Bool("shared", false, "generate code that can be linked into a shared library")
	Dynlink = fs.Bool("dynlink", false, "support references to Go symbols defined in other shared libraries")
	Linkshared = fs.Bool("linkshared", false, "generate code that will be linked against Go shared libraries")
	AllErrors = fs.Bool("e", false, "no limit on number of errors reported")
	SymABIs = fs.Bool("gensymabis", false, "write symbol ABI information to output file, don't assemble")
	Importpath = fs.String("p", obj.UnlinkablePkg, "set expected package import to path")
	Spectre = fs.String("spectre", "", "enable spectre mitigations in ` + "`list`" + ` (all, ret)")
	Std = fs.Bool("std", false, "building standard library")
	fs.Var(&D, "D", "predefined symbol with optional simple value -D=identifier=value; can be set multiple times")
	fs.Var(&I, "I", "include directory; can be set multiple times")
	fs.BoolVar(&DebugV, "v", false, "print debug output")
	fs.Var(objabi.NewDebugFlag(&DebugFlags, nil), "d", "enable debugging settings; try -d help")
	objabi.FlagcountFS(fs, "S", "print assembly and machine code", &PrintOut)
	DebugFlags.CompressInstructions = 1
}`
	patched := strings.Replace(next, oldInit, newInit, 1)
	if patched == next {
		return errors.New("asm flags init was not patched")
	}
	return os.WriteFile(path, []byte(patched), 0o644)
}

// patchLib applies the library-entry edits in patch.json. Each old string is
// upstream text after patchAll, with no module import path in it, so a modot
// replace of _upstream still rewrites compiler paths before these edits land.
func patchLib(dst string) error {
	body, err := os.ReadFile(filepath.Join("x", "toolchain", "go", "rewrite", "patch.json"))
	if err != nil {
		return err
	}
	var files map[string][]struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if err := json.Unmarshal(body, &files); err != nil {
		return err
	}
	rels := make([]string, 0, len(files))
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		path := filepath.Join(dst, filepath.FromSlash(rel))
		textb, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%w %s: %w", ErrPatchLibPatch, rel, err)
		}
		text := string(textb)
		for i, hunk := range files[rel] {
			if hunk.Old == "" {
				return fmt.Errorf("patch %s hunk %d: %w", rel, i, ErrPatchLibEmptyAnchor)
			}
			n := strings.Count(text, hunk.Old)
			if n != 1 {
				return fmt.Errorf("patch %s hunk %d: found %d %w", rel, i, n, ErrPatchLibAnchors)
			}
			text = strings.Replace(text, hunk.Old, hunk.New, 1)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// writeGenerated fills files that `go tool dist` writes into a GOROOT and that
// the git pin does not contain. The defaults match go1.27.0's dist output.
func writeGenerated(dst string) error {
	buildcfg := `// Code generated by x/toolchain/go/rewrite from go1.27.0 dist defaults. DO NOT EDIT.

package buildcfg

import "runtime"

const DefaultGO386 = ` + "`sse2`" + `
const DefaultGOAMD64 = ` + "`v1`" + `
const DefaultGOARM = ` + "`7`" + `
const DefaultGOARM64 = ` + "`v8.0`" + `
const DefaultGOMIPS = ` + "`hardfloat`" + `
const DefaultGOMIPS64 = ` + "`hardfloat`" + `
const DefaultGOPPC64 = ` + "`power8`" + `
const DefaultGORISCV64 = ` + "`rva20u64`" + `
const defaultGOEXPERIMENT = ` + "``" + `
const defaultGO_EXTLINK_ENABLED = ` + "``" + `
const defaultGO_LDSO = ` + "``" + `
const version = ` + "`go1.27.0`" + `
const defaultGOOS = runtime.GOOS
const defaultGOARCH = runtime.GOARCH
const DefaultGOFIPS140 = ` + "`off`" + `
const DefaultCGO_ENABLED = ""
`
	if err := os.WriteFile(filepath.Join(dst, "std", "buildcfg", "zbootstrap.go"), []byte(buildcfg), 0o644); err != nil {
		return err
	}
	objabi := "// Code generated by go tool dist; DO NOT EDIT.\n\npackage objabi\n"
	return os.WriteFile(filepath.Join(dst, "cmd", "objabi", "zbootstrap.go"), []byte(objabi), 0o644)
}

// packages is the amd64 and arm64 import closure of gc.Main and ld.Main, minus public
// standard library packages. Files behind a GOOS build tag count: copyPkg keeps
// them, and a consumer's go mod tidy reads their imports. Each pair is
// _upstream-relative then src-relative.
var packages = [][2]string{
	{"asm/internal/arch", "asm/arch"},
	{"asm/internal/asm", "asm/asm"},
	{"asm/internal/flags", "asm/flags"},
	{"asm/internal/lex", "asm/lex"},
	{"compile/internal/abi", "compile/abi"},
	{"compile/internal/abt", "compile/abt"},
	{"compile/internal/amd64", "compile/amd64"},
	{"compile/internal/arm64", "compile/arm64"},
	{"compile/internal/base", "compile/base"},
	{"compile/internal/bitvec", "compile/bitvec"},
	{"compile/internal/bloop", "compile/bloop"},
	{"compile/internal/compare", "compile/compare"},
	{"compile/internal/coverage", "compile/coverage"},
	{"compile/internal/deadlocals", "compile/deadlocals"},
	{"compile/internal/devirtualize", "compile/devirtualize"},
	{"compile/internal/dwarfgen", "compile/dwarfgen"},
	{"compile/internal/escape", "compile/escape"},
	{"compile/internal/gc", "compile/gc"},
	{"compile/internal/importer", "compile/importer"},
	{"compile/internal/inline", "compile/inline"},
	{"compile/internal/inline/inlheur", "compile/inline/inlheur"},
	{"compile/internal/inline/interleaved", "compile/inline/interleaved"},
	{"compile/internal/ir", "compile/ir"},
	{"compile/internal/liveness", "compile/liveness"},
	{"compile/internal/logopt", "compile/logopt"},
	{"compile/internal/loopvar", "compile/loopvar"},
	{"compile/internal/midway", "compile/midway"},
	{"compile/internal/noder", "compile/noder"},
	{"compile/internal/objw", "compile/objw"},
	{"compile/internal/pgoir", "compile/pgoir"},
	{"compile/internal/pkginit", "compile/pkginit"},
	{"compile/internal/rangefunc", "compile/rangefunc"},
	{"compile/internal/reflectdata", "compile/reflectdata"},
	{"compile/internal/rttype", "compile/rttype"},
	{"compile/internal/slice", "compile/slice"},
	{"compile/internal/ssa", "compile/ssa"},
	{"compile/internal/ssagen", "compile/ssagen"},
	{"compile/internal/staticdata", "compile/staticdata"},
	{"compile/internal/staticinit", "compile/staticinit"},
	{"compile/internal/syntax", "compile/syntax"},
	{"compile/internal/typebits", "compile/typebits"},
	{"compile/internal/typecheck", "compile/typecheck"},
	{"compile/internal/types", "compile/types"},
	{"compile/internal/types2", "compile/types2"},
	{"compile/internal/walk", "compile/walk"},
	{"link/internal/amd64", "link/amd64"},
	{"link/internal/arm64", "link/arm64"},
	{"link/internal/benchmark", "link/benchmark"},
	{"link/internal/ld", "link/ld"},
	{"link/internal/loadelf", "link/loadelf"},
	{"link/internal/loader", "link/loader"},
	{"link/internal/loadmacho", "link/loadmacho"},
	{"link/internal/loadpe", "link/loadpe"},
	{"link/internal/loadxcoff", "link/loadxcoff"},
	{"link/internal/sym", "link/sym"},
	{"cmd/archive", "cmd/archive"},
	{"cmd/bio", "cmd/bio"},
	{"cmd/codesign", "cmd/codesign"},
	{"cmd/cov/covcmd", "cmd/cov/covcmd"},
	{"cmd/dwarf", "cmd/dwarf"},
	{"cmd/gcprog", "cmd/gcprog"},
	{"cmd/goobj", "cmd/goobj"},
	{"cmd/hash", "cmd/hash"},
	{"cmd/macho", "cmd/macho"},
	{"cmd/obj", "cmd/obj"},
	{"cmd/obj/arm", "cmd/obj/arm"},
	{"cmd/obj/arm64", "cmd/obj/arm64"},
	{"cmd/obj/loong64", "cmd/obj/loong64"},
	{"cmd/obj/mips", "cmd/obj/mips"},
	{"cmd/obj/ppc64", "cmd/obj/ppc64"},
	{"cmd/obj/riscv", "cmd/obj/riscv"},
	{"cmd/obj/s390x", "cmd/obj/s390x"},
	{"cmd/obj/wasm", "cmd/obj/wasm"},
	{"cmd/obj/x86", "cmd/obj/x86"},
	{"cmd/objabi", "cmd/objabi"},
	{"cmd/pgo", "cmd/pgo"},
	{"cmd/quoted", "cmd/quoted"},
	{"cmd/src", "cmd/src"},
	{"cmd/sys", "cmd/sys"},
	{"cmd/telemetry/counter", "cmd/telemetry/counter"},
	{"std/abi", "std/abi"},
	{"std/bisect", "std/bisect"},
	{"std/buildcfg", "std/buildcfg"},
	{"std/coverage", "std/coverage"},
	{"std/exportdata", "std/exportdata"},
	{"std/goarch", "std/goarch"},
	{"std/goexperiment", "std/goexperiment"},
	{"std/goversion", "std/goversion"},
	{"std/lazyregexp", "std/lazyregexp"},
	{"std/pkgbits", "std/pkgbits"},
	{"std/platform", "std/platform"},
	{"std/profile", "std/profile"},
	{"std/race", "std/race"},
	{"std/runtime/gc", "std/runtime/gc"},
	{"std/saferio", "std/saferio"},
	{"std/strconv", "std/strconv"},
	{"std/syscall/unix", "std/syscall/unix"},
	{"std/types/errors", "std/types/errors"},
	{"std/unsafeheader", "std/unsafeheader"},
	{"std/xcoff", "std/xcoff"},
	{"telemetry/counter", "telemetry/counter"},
	{"telemetry/internal/counter", "telemetry/internal/counter"},
	{"telemetry/internal/mmap", "telemetry/internal/mmap"},
	{"telemetry/internal/telemetry", "telemetry/internal/telemetry"},
}

func copyPkg(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_test.s") {
			continue
		}
		if !strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return err
		}
		if strings.HasSuffix(name, ".go") {
			body, err = rewriteGo(name, body)
			if err != nil {
				return fmt.Errorf("%w%s: %w", ErrCopyPkg, name, err)
			}
		}
		if err := os.WriteFile(filepath.Join(dst, name), body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func rewriteGo(name string, src []byte) ([]byte, error) {
	src = dropCgoLdflag(src)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		next := rewriteImport(path)
		if next == path {
			continue
		}
		imp.Path.Value = strconv.Quote(next)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dropCgoLdflag removes //go:cgo_ldflag lines. The compiler accepts that
// directive in the standard library and in cgo-generated files, and rejects
// it in this module.
func dropCgoLdflag(src []byte) []byte {
	if !bytes.Contains(src, []byte("//go:cgo_ldflag")) {
		return src
	}
	lines := bytes.Split(src, []byte("\n"))
	kept := lines[:0]
	for _, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("//go:cgo_ldflag")) {
			continue
		}
		kept = append(kept, line)
	}
	return bytes.Join(kept, []byte("\n"))
}

func rewriteImport(path string) string {
	switch {
	case strings.HasPrefix(path, "cmd/compile/internal/"):
		return moduleRoot + "compile/" + strings.TrimPrefix(path, "cmd/compile/internal/")
	case strings.HasPrefix(path, "cmd/link/internal/"):
		return moduleRoot + "link/" + strings.TrimPrefix(path, "cmd/link/internal/")
	case strings.HasPrefix(path, "cmd/asm/internal/"):
		return moduleRoot + "asm/" + strings.TrimPrefix(path, "cmd/asm/internal/")
	case strings.HasPrefix(path, "cmd/internal/"):
		return moduleRoot + "cmd/" + strings.TrimPrefix(path, "cmd/internal/")
	case strings.HasPrefix(path, "golang.org/x/telemetry/"):
		return moduleRoot + "telemetry/" + strings.TrimPrefix(path, "golang.org/x/telemetry/")
	case strings.HasPrefix(path, "internal/"):
		return moduleRoot + "std/" + strings.TrimPrefix(path, "internal/")
	default:
		return path
	}
}

func patchAll(dst string) error {
	base := filepath.Join(dst, "compile", "base", "base.go")
	body, err := os.ReadFile(base)
	if err != nil {
		return err
	}
	oldExit := "func Exit(code int) {\n\tfor i := len(atExitFuncs) - 1; i >= 0; i-- {\n\t\tf := atExitFuncs[i]\n\t\tatExitFuncs = atExitFuncs[:i]\n\t\tf()\n\t}\n\tos.Exit(code)\n}"
	newExit := "// ExitSignal is the status Exit would have passed to os.Exit.\n// The toolchain driver recovers it and keeps the process.\ntype ExitSignal int\n\nfunc Exit(code int) {\n\tfor i := len(atExitFuncs) - 1; i >= 0; i-- {\n\t\tf := atExitFuncs[i]\n\t\tatExitFuncs = atExitFuncs[:i]\n\t\tf()\n\t}\n\tpanic(ExitSignal(code))\n}"
	next := strings.Replace(string(body), oldExit, newExit, 1)
	if next == string(body) {
		return ErrCompileBaseExitWasNotPatched
	}
	next = strings.Replace(next, "import (\n\t\"os\"\n)\n\n", "", 1)
	if err := os.WriteFile(base, []byte(next), 0o644); err != nil {
		return err
	}

	printGo := filepath.Join(dst, "compile", "base", "print.go")
	body, err = os.ReadFile(printGo)
	if err != nil {
		return err
	}
	next = strings.Replace(string(body), "\tos.Exit(2)\n", "\tExit(2)\n", 1)
	if next == string(body) {
		return ErrCompileErrorexitWasNotPatched
	}
	if err := os.WriteFile(printGo, []byte(next), 0o644); err != nil {
		return err
	}

	mainGo := filepath.Join(dst, "compile", "gc", "main.go")
	body, err = os.ReadFile(mainGo)
	if err != nil {
		return err
	}
	oldPanic := "\t\tif err == \"-h\" {\n\t\t\t// Force real panic now with -h option (hcrash) - the error\n\t\t\t// information will have already been printed.\n\t\t\tpanic(err)\n\t\t}\n"
	newPanic := "\t\tif err == \"-h\" {\n\t\t\tpanic(err)\n\t\t}\n\t\tif _, ok := err.(base.ExitSignal); ok {\n\t\t\tpanic(err)\n\t\t}\n"
	next = strings.Replace(string(body), oldPanic, newPanic, 1)
	if next == string(body) {
		return ErrCompileHandlepanicWasNotPatched
	}
	if err := os.WriteFile(mainGo, []byte(next), 0o644); err != nil {
		return err
	}

	utilGo := filepath.Join(dst, "link", "ld", "util.go")
	body, err = os.ReadFile(utilGo)
	if err != nil {
		return err
	}
	oldLink := "func Exit(code int) {\n\trunAtExitFuncs()\n\tos.Exit(code)\n}"
	newLink := "// ExitSignal is the status Exit would have passed to os.Exit.\n// The toolchain driver recovers it and keeps the process.\ntype ExitSignal int\n\nfunc Exit(code int) {\n\trunAtExitFuncs()\n\tpanic(ExitSignal(code))\n}"
	next = strings.Replace(string(body), oldLink, newLink, 1)
	if next == string(body) {
		return ErrLinkExitWasNotPatched
	}
	return os.WriteFile(utilGo, []byte(next), 0o644)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "rewrite: %v\n", err)
	os.Exit(1)
}
