# lewkit

[![Go Reference](https://pkg.go.dev/badge/github.com/lewtec/lewkit.svg)](https://pkg.go.dev/github.com/lewtec/lewkit)
[![Go version](https://img.shields.io/github/go-mod/go-version/lewtec/lewkit)](go.mod)
[![Build](https://img.shields.io/github/actions/workflow/status/lewtec/lewkit/autorelease.yaml?branch=main&label=build)](https://github.com/lewtec/lewkit/actions/workflows/autorelease.yaml)
[![Release](https://img.shields.io/github/v/release/lewtec/lewkit)](https://github.com/lewtec/lewkit/releases)
[![License](https://img.shields.io/badge/license-GPL-blue.svg)](LICENSE.md)

lewkit is a reuse library of Go primitives. The module path is `github.com/lewtec/lewkit`. The module requires Go 1.27. Packages build with `CGO_ENABLED=0`.

Rushed code is not sustainable long term and this is our tool of reuse.

```bash
go get github.com/lewtec/lewkit@main
```

`@latest` selects the highest tag. That tag can lag `main`.

The program is `cmd/lewkit`. [SPEC.md](SPEC.md) names the owner package for a new type. The license text is [LICENSE.md](LICENSE.md).

`x/ffi` and `x/ui` have no Go package. An import of either path fails. [SPEC.md](SPEC.md) lists `x/ui/tui` and `x/ui/web`. Both packages are absent.

## Packages

Prefix every path with `github.com/lewtec/lewkit/`.

### Files

| Path | API |
| --- | --- |
| `x/path` | `Path`, `New`, `Open`. A `Path` is a slash name. `Open` takes the OS directory. |
| `x/path/pick` | `Predicate`, `Match`, `Glob`, `Prune`, `And`, `Or`, `Not`. |
| `x/fs` | `Files`, `Walk`, `Filter`, `Copy`, `New`, `StripTopDirectory`. `Walk` reads an `io/fs`. `Copy` writes a listing. `New` indexes a listing. `StripTopDirectory` returns an `io/fs` with one leading directory removed. |
| `x/fs/compose` | `New`, `Add`, `Merge`, `All`, `Squash`, `FS`, `Mount`, `Parse`, `Register`. A symlink is a `link` slot. `Squash` turns `name.d.tmpl/` into `lines` slots. `FS` encodes the tree. |
| `x/fs/tar` | `Open`, `Files`. A tar archive as `io/fs`. |
| `x/fs/zip` | `Open`. A ZIP archive as `io/fs`. |
| `x/fs/squashfs` | `Open`. A SquashFS image as `io/fs`. |
| `x/fs/udf` | `Open`. A UDF volume as `io/fs`. |
| `x/fs/wim` | `Open`. One WIM image as `io/fs`. |
| `x/compression` | `Codec`, `Detect`, `Register`, `ByExtension`, `ByMagic`, `New`. |
| `x/compression/gzip` | gzip `Codec`. `brotli`, `lz4`, `zstd`, `xz`, and `bzip2` match this shape. |
| `x/compression/prelude` | Blank-import. Registers gzip, brotli, lz4, zstd, xz, and bzip2. |
| `x/db` | `Arg`, `FromURL`, `Open`, `Migrate`, `Value`, `Conn.Tx`, `Conn.Queries`. |
| `x/db/sqlite` | Blank-import. Schemes `sqlite`, `sqlite3`, `file`, a bare path, and `:memory:`. |
| `x/db/postgres` | Blank-import. Scheme `postgres`. |
| `x/db/generate` | Called by `lewkit generate db`. Writes `Queries` and `DBArg`. |
| `x/io` | `Mkdirp`. |
| `x/io/atomic` | `NewOperation`, `Commit`, `Rollback`, `WriteFileFunction`, `WriteString`. |
| `x/text` | `LineIndex`, `NewLineIndex`. UTF-8 byte offset to line and column. |

### Process

| Path | API |
| --- | --- |
| `cmd/lewkit` | The `lewkit` program. |
| `cmd/lewkit/experiments` | Demo commands. Only `cmd/lewkit` imports this package. |
| `report` | `Reporter`, `RegisterReporter`, `Report`, `Must`. |
| `report/sentry` | Sentry `Reporter`. |
| `x/logging` | `NewHandler`. Level letter, message, then `key=value`. Color on a terminal. |
| `x/cmd` | `Parse`, `App`. Struct fields become commands and flags. This package writes bash completion. |
| `x/taskgroup` | `Session`, `New`, `Go`, `Map`, `Each`, `List`, `WithSession`, `GoIsolated`. Pools are IO, CPU, and internet. |
| `x/taskgroup/progress` | Bubbletea view of a `Session`. |
| `x/thread` | `Run`, `Bind`, `Do`, `Go`, `Loop`. `Run` starts the call from `main`. |
| `x/event` | `Bus`, `New`, `Subscribe`, `Publish`, `CreateTimer`, `FPS`. |
| `x/future` | `Future`, `NewFuture`, `Get`, `Peek`, `State`. |
| `x/dotfiles` | `Root`. First existing directory among the Codespaces share, `~/.dotfiles`, and `/etc/.dotfiles`. |
| `x/git` | `Git`, `Info`, `Worktree`, `Resolve`. Checkouts, branches, and linked worktrees. |
| `x/herdr` | `Client`, `RepoBranch`, `Reorder`. Herdr workspace layout. |
| `x/singleton` | `NewSingleton`, `Get`, `MustGet`. |
| `x/profile` | `Directory`, `Address`, `Handler`. pprof to a directory or HTTP. |
| `x/http/middleware` | `SPA`. Serves an `fs.FS` with the goftpd SPA rules. A miss goes to the next handler. |
| `x/http/asset` | `Mount`, `Register`. Serves registered files under `/__lewkit__/`. |
| `x/http/asset/htmx` | Blank-import. Registers htmx and renders `Load`. `jquery`, `tailwindcss`, and `sakuracss` match this shape. |
| `x/http/asset/prelude` | Blank-import. Registers htmx, tailwindcss, jquery, and sakuracss. |
| `x/release` | `Version`, `PrintVersion`, `Platform`. `lewkit --version` prints `Version`. |
| `x/test` | Helpers for process globals, closers, iterators, and readers. |
| `x/auth` | `HashPassword`, `HashPasswordCost`, `CheckHashedPassword`. Bcrypt. |
| `x/generate` | Helpers shared by the generator packages. |
| `x/generate/prelude` | Writes a blank-import file from each `root.go`. |
| `x/generate/protobuf` | Writes Go from one `.proto` file. `protoc` comes from `x/tool`. A `workspaced.lock.json` pin selects the version. |
| `x/tool` | `Open`, `Ensure`, `Install`, `Resolve`. A spec is `backend:ref@version`. The caller owns the store directory. |
| `x/tool/github` | GitHub Releases backend. |
| `x/tool/mise` | mise backend. |
| `x/tool/registry` | Short-name backend. Curated names live in `x/tool/registry/applications`. |
| `x/tool/prelude` | Blank-import. Registers GitHub, mise, the short-name registry, and the curated names. |

### Tensors

| Path | API |
| --- | --- |
| `x/ndarray` | `Tensor`, `Eval`, `Open`, `New`, `Zeros`, `Ones`, `Full`, `Rand`, `Const`, `Coord`, `Shape`. One fused kernel of 21 ALU ops. |
| `x/ndarray/nn` | `Convolution2D`, `MaximumPool2D`, `AveragePool2D`, `MatrixMultiply`, `Linear`. |
| `x/ndarray/onnx` | `Load`, `LoadBytes`, `FunctionOf`, `Apply`, `ApplyInputs`. |
| `x/ndarray/image` | `Fill`, `Eval`, `Raster`, `Write`, `RGBA`. Pixels are `(h, w, 4)`. |
| `x/image` | `Triangle`, `TriangleTurn`, `Label`, `CopyRGBA`, `CenterSquare`, `Face`. |
| `x/graph` | `Graph`, `DOT`, `Mermaid`. |
| `x/ui/gui` | `Model`, `Node`, `Box`, `Row`, `Column`, `Stack`, `Text`, `Run`, `Open`, `Tick`, `Every`. |

`ndarray.Open` returns the highest-weight `Evaluator`. Blank-import `x/driver/ndeval` or `x/driver/prelude` first.

`Model` is `Init`, `Update`, `View`. `View` returns a `Node`. `Run` paints a window the caller opened. `Open` calls `window.Open`, then `Run`. Layout types are `Box`, `Row`, `Column`, and `Stack`.

### Host and bindings

| Path | API |
| --- | --- |
| `x/driver` | `Register`, `List`, `Get`, `With`, `WithResult`, `SetWeights`, `Doctor`. |
| `x/driver/prelude` | Blank-import. Registers the window backends, `x/driver/vulkan`, `x/driver/ndeval`, `x/driver/httpclient`, and `x/driver/fetchurl`. |
| `x/driver/httpclient` | `Client`. A request inside a taskgroup session is an Internet task. The native client is `x/driver/httpclient/native`. |
| `x/driver/fetchurl` | `Fetch`. The native driver calls `github.com/fetchurl/fetchurl` with that client, so the download is the same Internet task. |
| `x/driver/window` | `Open`, `Frame`, `Front`, `Draw`, `Fit`, `Present`, `Animate`, `Drive`, `Subscribe`. |
| `x/driver/window/cocoa` | macOS backend. `Open` runs on the process main thread. Call `thread.Run` from `main`. |
| `x/driver/window/win32` | Windows backend. |
| `x/driver/window/x11` | X11 backend. |
| `x/driver/window/mem` | In-memory backend for tests. |
| `x/driver/vulkan` | Facade. `Open`, `List`, `Buffer`, `Compile`, `Begin`. Re-exports `Buffer`, `Shader`, and `Cmd`. |
| `x/driver/ndeval` | `Evaluator` factories. The CPU factory registers at init. Vulkan wraps the selected GPU. |
| `x/disasm` | Facade for `x/ffi/wasm/capstone`. `Open`, `Engine.Iter`, `DecodeHex`, `ReadText`, `OpenObject`, `FormatInstruction`. |
| `x/ffi/native` | `Open`, `Func`, `Symbol`, `Register`. Loads a shared library without cgo. |
| `x/ffi/native/vulkan` | Binding. `Open`, `List`, `Buffer`, `Alloc`, `Shader`, `Compile`, `Run`, `Begin`. |
| `x/ffi/wasm` | `Compile`, `Compiled.Instantiate`. wazero, with WASI and optional Emscripten. |
| `x/ffi/wasm/capstone` | Binding. `Open`, `Handle`. |
| `x/ffi/wasm/glsl` | `Compile`, `Load`, `IsSPIRV`. `Compile` turns GLSL into SPIR-V. `Load` keeps a SPIR-V buffer as-is. |

`window.Subscribe` yields `Resize`, `Expose`, `Close`, `Pointer`, `Scroll`, and `Key`.

`x/driver/vulkan` keeps the libvulkan `Device` private. `x/driver/ndeval` compiles a kernel with `x/ffi/wasm/glsl` and runs it on that facade.

`x/ffi/native/vulkan` `Cmd` methods are `Bind`, `Push`, `Dispatch`, `Copy`, `Barrier`, `Submit`, `Wait`, and `Abort`. `Shader` takes SPIR-V.

`x/disasm` reads ELF, PE, and Mach-O through `x/ffi/wasm/capstone`.

## Commands

Build the program from a clone with `go build -o lewkit ./cmd/lewkit`.

Global flags are `-h`, `-v`, `--version`, `--pprof`, and `--sentry-dsn`. `SENTRY_DSN` sets the same DSN.

### Program

| Command | Result |
| --- | --- |
| `lewkit doctor` | Each driver interface and the implementation `Get` selected. |
| `lewkit herdr reorder [REPO:BRANCH...]` | Nest linked worktrees, park a feature branch off the main checkout, and order workspaces. |
| `lewkit disasm hex HEX` | Instructions for a hex byte string. |
| `lewkit disasm raw PATH` | Instructions for a raw byte file. `PATH` `-` reads stdin. |
| `lewkit disasm file PATH` | One text section from an ELF, PE, or Mach-O file. |
| `lewkit generate db DIR` | sqlc packages, a `Queries` interface, and `DBArg`. |
| `lewkit generate prelude DIR [OUT]` | Blank-import prelude from each `root.go` under `DIR`. |
| `lewkit generate protobuf FILE` | Go source for a `.proto` file. |
| `lewkit completion` | The bash `complete -C` line for this program. |

`lewkit disasm` flags are `--architecture` (default `x86`), `--mode` (default `64`), and `--syntax` (default `default`). Shared flags are `--address`, `--count`, and `--skip-data`. A `--count` of `0` prints every instruction. `lewkit disasm file` also takes `--section`.

`lewkit generate db` reads `sqlite/` and `postgres/` under `DIR`. An omitted `OUT` on `generate prelude` writes stdout. `generate protobuf` takes `--package` when the file has no `go_package`.

`lewkit herdr reorder` reads the running Herdr session. Each `REPO:BRANCH` ensures a main workspace and a linked worktree at `~/.grok/worktrees/<slug>/<branch>`. A slash in the branch is a hyphen in the directory name. The dotfiles root from `x/dotfiles` sorts first. Steps log through slog and the progress view. The final order is a progress tree: each worktree sits under its main checkout. A workspace row is the label, then the checkout path.

### Demos

| Command | Result |
| --- | --- |
| `lewkit experiments demo tasks` | Progress bars, logs, pools, and dependencies. |
| `lewkit experiments demo plain` | The `tasks` schedule with no progress view. |
| `lewkit experiments demo nested` | `GoIsolated` around child tasks. |
| `lewkit experiments demo loop` | Five steps and a moving bar. |
| `lewkit experiments demo map` | `Map` over a list under one bar. |
| `lewkit experiments demo many` | 256 `Map` items. The view calls `List(n)`. |
| `lewkit experiments demo tree` | A deep tree: release, then fetch, compile, and package. |
| `lewkit experiments demo lines` | Three rows that rewrite until a newline. |
| `lewkit experiments demo rsync` | Parallel fake transfers. Each transfer rewrites one row. |
| `lewkit experiments window triangle` | RGB triangle, one turn per second. Flags `--width` and `--height`. |
| `lewkit experiments window perlin` | Animated Perlin noise. Flags `--width` and `--height`. |
| `lewkit experiments window compute [SHADER]` | Default shader `example.comp`. `SHADER` is a `.spv` or `.comp` path. |
| `lewkit experiments window scroll` | Rounded translucent boxes in a loop. Flags `--width` and `--height`. |
| `lewkit experiments window notepad` | An editor. The buffer stays in memory. Flags `--width` and `--height`. |
| `lewkit experiments window counter` | Two buttons that add and subtract an integer. Flags `--width` and `--height`. |

`lewkit experiments window compute --smoke` prints a 4-byte probe. The probe needs a Vulkan compute device.

`demo` commands run in the terminal. `window` commands call `window.Open`.
