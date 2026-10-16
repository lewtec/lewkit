# x/takemake

Each child is one walkable model. Callers compose it with a jennifer-style fluent API. `Take` loads bytes into that model. `Make` renders it. A composed value and a taken value are the same type, so the proof is `Take`, `Make`, `Take`, and the two values match.

The parent directory has no Go package. An import of `github.com/lewtec/lewkit/x/takemake` fails.

```
x/takemake/elf
x/takemake/pe
x/takemake/macho
x/takemake/golang
x/takemake/c
x/takemake/proto
x/takemake/glsl
x/takemake/sql
x/takemake/sql/schema
x/takemake/tar
x/takemake/zip
x/takemake/makefile
x/takemake/ninja
x/takemake/graphviz
x/takemake/iso
x/takemake/squashfs
x/takemake/udf
x/takemake/wim
x/takemake/onnx
x/takemake/sarif
x/takemake/dotenv
```

A session owns the directories in its node and leaves the others alone. The scaffold compiles and returns `errs.ErrScaffold` from `Take` and `Make`. Replace that body. Closed sets are `//lewkit:enum`, and the package `enum.go` holds only the package clause and `//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum`.

## v1

A session adds code only inside the directories it owns, including a nested package. It does not edit another takemake directory, the parent, or any package outside `x/takemake`. Importing a dependency that already exists is fine. A duplicated helper is fine. A shared package for that helper waits until the models exist.

Tests live in the owned packages. `Take`, then `Make`, then `Take` again, and the two values match on the fields named below. The scaffold test is the floor. The round trip, and the fields that must survive it, need their own tests.

Wiring `x/disasm`, `x/fs`, `x/workflow`, `x/generate`, `x/graph`, `x/ndarray`, `x/text/report`, and `x/sops` to these files is a later pass.

## API

Methods return the receiver and store a real child on the parent. `Take` returns that same root.

```go
func Take(r io.Reader) (*File, error)
func (f *File) Make(w io.Writer) error
```

`golang` and the SQL language use `NewFile`. The others use `New`. `schema.Make` returns a `*sql.File` of `CreateTable` statements. It does not write bytes.

```go
f := golang.NewFile("main")
f.Func().Id("main").Params().Block(
    golang.Qual("fmt", "Println").Call(golang.Lit("hi")),
)
```

After that chain, `f` has one function named `main`. Jennifer may print behind `golang.Make`. Callers compose `golang.File`. The package path is `golang` because `package go` is a keyword.

## SQL is two models

`x/takemake/sql` is the statement log: `CreateTable`, `AlterTable`, `DropTable`, `CreateIndex`, `Select`, `Insert`, `Update`, and `Delete`. It does not import `schema`.

`x/takemake/sql/schema` is the catalog after those statements run. `Apply` folds the log in order. `DropTable` removes a table. `AlterTable` changes a column. `schema.Make` emits the `CreateTable` snapshot. Applying that snapshot to an empty catalog reproduces the tables, columns, keys, and foreign keys.

```go
up := sql.NewFile()
up.CreateTable("artifact").
    Column("name", sql.Text()).PrimaryKey().
    Column("format", sql.Text()).NotNull().
    Column("entry", sql.Integer()).NotNull()

db := schema.New()
db.Apply(up)
db.Table("artifact").Column("name")
```

The migration to fold first is `x/disasm/decomp/schema/sqlite/migrations/000001_project.up.sql`. Dialect is an enum on the SQL file. The catalog records it.

## What each file holds

`elf.File` holds class, endian, file kind, machine, entry, sections, and program headers. A section has a name, kind, flags, address, alignment, and bytes. A symbol hangs off the section that defines it. A reloc is an offset, a symbol, a kind, and an addend. A segment names the sections it covers.

```go
elf.New().Class64().Little().Machine(elf.X86_64)
f.Section(".text").Alloc().Exec().Addr(0x1000).Bytes(code).Symbol("main").Func().Global()
```

This replaces `disasm.ELF64` in `x/disasm/elf64.go`, which writes one `.text` and drops the symbol table. The proof is that sections and symbols survive the round trip.

`pe.File` holds machine, subsystem, image base, entry, sections, imports, and exports.

```go
pe.New().Machine(pe.Amd64)
f.Section(".text").Exec().Read().Bytes(code)
f.Import("kernel32").Symbol("GetProcAddress")
```

`macho.File` holds CPU, file kind, and load commands in order: segment, section, symbol, dylib, and entry.

```go
macho.New().CPU(macho.Amd64).Dylib()
f.Segment("__TEXT").Section("__text").Bytes(code).Symbol("_main")
```

`golang.File` holds the package name, imports, and declarations (`Const`, `Var`, `Type`, `Func`). Expressions are identifiers, selectors, calls, literals, composites, and the unary and binary forms. Types are named types, pointers, slices, arrays, maps, structs, interfaces, and function types. The chain is `Func`, `Type`, `Var`, `Const`, `Id`, `Qual`, `Lit`, `Params`, `Block`, and `Call`.

`c.File` is one C translation unit: includes and declarations (`Type`, `Const`, `Func`, `Object`). A function keeps its body, so a `.c` file round-trips the statements and not only the signature. A header is that same file with no bodies. A type is void, bool, integer, float, pointer, array, function, struct, union, or enum. Size, alignment, and field offset live on the type. `Take` parses C. `Make` prints C. A projection to `golang.File` is separate. `x/generate/header` `Unit` is the declaration model this replaces for headers; it does not hold a body.

`proto.File` holds syntax, package, imports, options, messages, enums, and services. A field is a name, a number, a type, and a label. `Make` prints `.proto`. The Go projection emits a `golang.File`. `x/generate/protobuf` runs `protoc` today.

`glsl.File` holds a stage, a version, uniforms, inputs, outputs, and functions. `Take` and `Make` are GLSL text. SPIR-V stays the compile product of `x/ffi/wasm/glsl`.

```go
glsl.New(glsl.Fragment).Version(450)
s.Uniform("color", glsl.Vec4())
s.Func("main").Returns(glsl.Void()).Block(...)
```

`tar.File` and `zip.File` are ordered entries. An entry is a name, a mode, a modification time, and either bytes or a symlink target. `x/fs/tar` and `x/fs/zip` only open today.

`makefile.File` holds variables, rules, and functions. A rule is targets, prerequisites, and a recipe. `Take` parses a Makefile. `Make` prints it. `x/workflow/make` lowers that file to `workflow.Graph`.

`ninja.File` holds variables, rules, and build edges. An edge is outputs, a rule, inputs, and bindings. `Take` parses the file. `Make` prints it. `x/workflow/ninja` lowers edges to `workflow.Graph`.

`graphviz.File` holds graphs, nodes, edges, and attributes. `Take` parses Graphviz source. `Make` prints it. `x/graph.Graph` is a projection. `Graph.DOT` prints a small digraph today.

`iso.File` is an ISO 9660 directory. An entry is a name, a mode, a time, and bytes. A zero catalog is a data image. `Platform`, `Emulation`, or `Boot` opts into El Torito: the boot catalog and a FAT16 boot image. `x/fs/iso.Write` builds that bootable image today.

`squashfs.File`, `udf.File`, and `wim.File` are file trees. An entry is a name, a mode, a time, and bytes or a symlink target. `x/fs/squashfs`, `x/fs/udf`, and `x/fs/wim` only open today. UDF is its own volume. A file named `.iso` often holds one, and `x/fs/udf` reads that volume. `iso.File` does not.

`onnx.File` holds inputs, outputs, initializers, and nodes. `Take` reads a ModelProto. `Make` writes it. `x/ndarray/onnx` evaluates the graph.

`sarif.File` is one SARIF 2.1.0 log: runs, results, rules, and fixes. `Take` parses the JSON. `Make` prints it. `x/text/report.WriteSARIF` prints findings today.

`dotenv.File` is assignments in order. A comment and the original quoting stay on the assignment. `Take` parses the text. `Make` prints it. `x/sops.ParseEnv` drops comments, and `Env.Strings` is not that text.

## Leave these where they are

`x/disasm` keeps Capstone. A later pass reads section bytes from `elf.File`, `pe.File`, or `macho.File`. `x/disasm/decomp` keeps the lift. `x/toolchain/go` keeps linking. `x/generate/clib` keeps `MakeShared`, `MakeDLL`, and the Mach-O dylib rewrite.

SVG, gemtext, WAV, PNG, JPEG, ICO, ICNS, the compression codecs, and the AppImage trailer already round-trip. Mermaid stays a printer on `x/graph`. The rustc diagnostic text stays a printer on `x/text/report`. JSON, TOML, YAML, INI, and XML stay encodings of one `compose.File`.
