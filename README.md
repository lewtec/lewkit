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
| `x/fs` | `Files`, `Walk`, `Filter`, `Copy`, `Index`, `New`, `Lookup`, `StripTopDirectory`. `Walk` reads an `io/fs`. `Copy` writes a listing. `Index` decorates a flat listing. `New` builds that index. `Lookup` walks a filesystem that already has directories. `StripTopDirectory` returns an `io/fs` with one leading directory removed. |
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
| `x/io/atomic` | `NewOperation`, `Commit`, `Rollback`, `WriteFileFunction`, `WriteString`. |
| `x/text` | `LineIndex`, `NewLineIndex`. UTF-8 byte offset to line and column. |

### Process

| Path | API |
| --- | --- |
| `cmd/lewkit` | The `lewkit` program. |
| `report` | `Reporter`, `RegisterReporter`, `Report`, `Must`. |
| `report/sentry` | Sentry `Reporter`. |
| `x/logging` | `NewHandler`. Level letter, message, then `key=value`. Color on a terminal. |
| `x/cmd` | `Parse`, `App`. Struct fields become commands and flags. This package writes bash completion. |
| `x/taskgroup` | `Session`, `New`, `Go`, `Map`, `Each`, `List`, `WithSession`, `GoIsolated`. Pools are IO, CPU, and internet. |
| `x/taskgroup/progress` | Bubbletea view of a `Session`. |
| `x/driver/thread` | `Run`, `Bind`, `Do`, `Go`, `Loop`. `Run` starts the call from `main`. |
| `x/event` | `Bus`, `New`, `Subscribe`, `Publish`, `CreateTimer`, `FPS`. |
| `x/dotfiles` | `Root`. First existing directory among the Codespaces share, `~/.dotfiles`, and `/etc/.dotfiles`. |
| `x/git` | `Git`, `Info`, `Worktree`, `Resolve`. Checkouts, branches, and linked worktrees. |
| `x/singleton` | `NewSingleton`, `Get`, `MustGet`. |
| `x/profile` | `Directory`, `Address`, `Handler`. pprof to a directory or HTTP. |
| `x/http/middleware` | `SPA`. Serves an `fs.FS` with the goftpd SPA rules. A miss goes to the next handler. |
| `x/http/asset` | `Mount`, `Register`. Serves registered files under `/__lewkit__/`. |
| `x/http/asset/htmx` | Blank-import. Registers htmx and renders `Load`. `jquery`, `tailwindcss`, and `sakuracss` match this shape. |
| `x/http/asset/prelude` | Blank-import. Registers htmx, tailwindcss, jquery, and sakuracss. |
| `x/release` | `Version`, `AppID`, `ValidateAppID`, `Name`, `PrintVersion`, `Platform`. `lewkit --version` prints `Version`. The reverse-domain id is the `-X` stamp `x/release.appID`, or `LEWKIT_APP_ID` when the stamp is empty. The short name is the `-X` stamp `x/release.name`, or `LEWKIT_NAME`, or the built-in default. |
| `x/driver/bundle` | `Resolve`, `SharePath`. Data, cache, config, and the web profile for `AppID`. |
| `x/build` | `Job`, `Host`. `lewkit release build` writes one binary archive for this process's GOOS and GOARCH. `--goos` and `--goarch` override that. `--app` writes the host instead: a macOS `.app`, an Android APK, or an iOS `.app`. `lewkit release run` takes the same flags as `lewkit release build`, builds that artifact, and runs it. Arguments after `--` go to that program. |
| `x/app` | `Web`, `GUI`, `Open`, `Run`. An app is the windows of one process. Each window is a web handler or a GUI model, and it may open another. `Run` returns when the last window closes. `LEWKIT_NO_UI` or `ELETROCROMO_NO_UI` serves the first web handler on a loopback port. |
| `x/test` | Helpers for process globals, closers, iterators, and readers. |
| `x/auth` | `HashPassword`, `HashPasswordCost`, `CheckHashedPassword`. Bcrypt. |
| `x/generate` | Helpers shared by the generator packages. |
| `x/generate/prelude` | Writes one blank-import prelude per directory that contains a descendant `root.go`. |
| `x/generate/protobuf` | Writes Go from one `.proto` file. `protoc` comes from `x/tool`. A `modot.lock.json` pin selects the version. |
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
| `x/image` | `Label`, `CopyRGBA`, `CenterSquare`, `Face`. |
| `x/graph` | `Graph`, `DOT`, `Mermaid`. |
| `x/ui/gui` | `Model`, `Node`, `Box`, `Row`, `Column`, `Stack`, `Text`, `Run`, `Pick`, `Tick`, `Every`. |

`ndarray.Open` returns the highest-weight `Evaluator`. Blank-import `x/driver/ndeval` or `x/driver/prelude` first.

`Model` is `Init`, `Update`, `View`. `View` returns a `Node`. `Run` paints a window the caller opened. `x/app` opens that window. Layout types are `Box`, `Row`, `Column`, and `Stack`.

### Host and bindings

| Path | API |
| --- | --- |
| `x/driver` | `Register`, `List`, `Get`, `With`, `WithResult`, `SetWeights`, `Doctor`. |
| `x/driver/prelude` | Blank-import. Registers every driver. A group with its own descendant `root.go` files has a prelude too, for example `x/driver/webview/prelude`. |
| `x/driver/httpclient` | `Client`. A request inside a taskgroup session is an Internet task. The native client is `x/driver/httpclient/native`. |
| `x/driver/fetchurl` | `Fetch`. The native driver calls `github.com/fetchurl/fetchurl` with that client, so the download is the same Internet task. |
| `x/sound` | `Format`, `Mixer`, `Mix`, `Pipeline`, `Decode`, `Register`, `WriteWAV`, `ReadWAV`. `Decode` picks a decoder by extension or magic. |
| `x/sound/mp3` | MP3 decoder. Blank-imported by `x/sound/prelude`. |
| `x/sound/ogg` | Ogg Vorbis decoder. Blank-imported by `x/sound/prelude`. |
| `x/sound/prelude` | Blank-import. Registers MP3 and Ogg Vorbis. WAV registers with `x/sound`. |
| `x/driver/audio_play` | `Open`, `Sinks`. `Open` is an `io.WriteCloser` for one sink. An empty sink is the server default. |
| `x/driver/audio_play/pulse` | Linux playback through libpulse-simple. `Sink` is the PulseAudio sink name. PipeWire serves that API. |
| `x/driver/audio_play/winmm` | Windows playback through waveOut. `Sink` is a device index or the endpoint name. An empty sink is `WAVE_MAPPER`. |
| `x/driver/audio_play/coreaudio` | macOS playback through AudioQueue. `Sink` is a device UID or the display name. |
| `x/driver/audio_play/mem` | Records PCM. Incompatible unless `LEWKIT_ENABLE_MEMORY_DRIVER` is set. |
| `x/driver/window` | `Open`, `Frame`, `Front`, `Draw`, `Fit`, `Present`, `Animate`, `Drive`, `Subscribe`. |
| `x/driver/window/cocoa` | macOS backend. `Open` runs on the process main thread. Call `thread.Run` from `main`. |
| `x/driver/window/win32` | Windows backend. |
| `x/driver/window/x11` | X11 backend. |
| `x/driver/window/mem` | In-memory backend for tests. |
| `x/driver/vulkan` | Facade. `Open`, `List`, `Buffer`, `Compile`, `Begin`. Re-exports `Buffer`, `Shader`, and `Cmd`. |
| `x/driver/ndeval` | `Evaluator` factories. The CPU factory registers at init. Vulkan wraps the selected GPU. |
| `x/disasm` | Facade for `x/ffi/wasm/capstone`. `Open`, `Engine.Iter`, `DecodeHex`, `ReadText`, `OpenObject`, `FormatInstruction`. |
| `x/ffi/native` | `Open`, `Func`, `Symbol`, `Register`. Loads a shared library without cgo. |
| `x/ffi/native/android` | `JavaVMs`, `OnLooper`. libnativehelper and libandroid. |
| `x/ffi/android` | Binder session. `Open`, `Client`. |
| `x/ffi/jni` | Java calls. `Bind`, `CallStatic`, `New`, `Class`, `StaticField`, `Field`, `Proxy`. |
| `x/ffi/native/vulkan` | Binding. `Open`, `List`, `Buffer`, `Alloc`, `Shader`, `Compile`, `Run`, `Begin`. |
| `x/ffi/wasm` | `Compile`, `Compiled.Instantiate`. wazero, with WASI and optional Emscripten. |
| `x/ffi/wasm/capstone` | Binding. `Open`, `Handle`. |
| `x/ffi/wasm/glsl` | `Compile`, `CompileStage`, `Load`, `IsSPIRV`, `Hash`, `RegisterHash`, `Lookup`. A registered shader hash returns SPIR-V. A missing hash runs `glslc` when it is on `PATH`, then the embedded glslang. `Load` keeps a SPIR-V buffer as-is. |

`window.Subscribe` yields `Resize`, `Expose`, `Close`, `Pointer`, `Scroll`, and `Key`.

`x/driver/vulkan` keeps the libvulkan `Device` private. `x/driver/ndeval` loads a kernel through `x/ffi/wasm/glsl` and runs it on that facade. A registered hash is stored SPIR-V. Anything else compiles, with `glslc` when it is on `PATH` and the embedded glslang otherwise.

`x/ffi/native/vulkan` `Cmd` methods are `Bind`, `Push`, `Dispatch`, `Copy`, `Barrier`, `Submit`, `Wait`, and `Abort`. `Shader` takes SPIR-V.

`x/disasm` reads ELF, PE, and Mach-O through `x/ffi/wasm/capstone`.

## Commands

Build the program from a clone with `go build -o lewkit ./cmd/lewkit`.

Global flags are `-h`, `-v`, `--version`, `--pprof`, and `--sentry-dsn`. `SENTRY_DSN` sets the same DSN.

### Program

| Command | Result |
| --- | --- |
| `lewkit doctor` | Each driver interface and the implementation `Get` selected. |
| `lewkit disasm hex HEX` | Instructions for a hex byte string. |
| `lewkit disasm raw PATH` | Instructions for a raw byte file. `PATH` `-` reads stdin. |
| `lewkit disasm file PATH` | One text section from an ELF, PE, or Mach-O file. |
| `lewkit generate db DIR` | sqlc packages, a `Queries` interface, and `DBArg`. |
| `lewkit generate prelude DIR [OUT]` | One blank-import prelude per directory under `DIR` that contains a descendant `root.go`. `OUT` is the file for `DIR`. |
| `lewkit generate protobuf FILE` | Go source for a `.proto` file. |
| `lewkit generate shader DIR` | One `spirv_gen.go` per Go package under `DIR` that owns a `.vert`, `.frag`, or `.comp` file. The file maps that shader hash to SPIR-V. |
| `lewkit completion` | The bash `complete -C` line for this program. |

`lewkit disasm` flags are `--architecture` (default `x86`), `--mode` (default `64`), and `--syntax` (default `default`). Shared flags are `--address`, `--count`, and `--skip-data`. A `--count` of `0` prints every instruction. `lewkit disasm file` also takes `--section`.

`lewkit generate db` reads `sqlite/` and `postgres/` under `DIR`. An omitted `OUT` on `generate prelude` writes the prelude for `DIR` to stdout and does not write the nested preludes. `generate protobuf` takes `--package` when the file has no `go_package`. `generate shader` compiles with the embedded glslang, so the committed bytes do not depend on a host `glslc`. `DIR` defaults to the working directory. Packages that ship shaders carry `//go:generate go run ... generate shader .`, the same place a templ package carries `//go:generate go tool templ generate`.

### Examples

Each directory under `examples/` is one program. `eletrocromo.json` sits next to the Go main. Run a program from the repository root:

```bash
go run ./cmd/lewkit release run --config ./examples/basic/eletrocromo.json
```

A plain `go run` of an example has no release stamp and panics.

The progress view tracks the build. The program starts after that view and writes its output.

| Directory | Result |
| --- | --- |
| `examples/basic` | One-line web handler. |
| `examples/counter` | Server-rendered increment form. |
| `examples/ticker` | Adds one every second. The page reads that count. |
| `examples/inbox` | URL and file opens. Notes go under the app dirs. |
| `examples/astro` | Astro page in a web window. Build the embed first. See `examples/astro/README.md`. |
| `examples/host` | Host drivers, brightness, and the clipboard. |
| `examples/drivers` | Driver panels. Opens the triangle window. |
| `examples/tasks` | Progress bars, logs, pools, and dependencies. |
| `examples/plain` | Fetch, process, and write. |
| `examples/nested` | Isolated child tasks inside a bundle. |
| `examples/loop` | Five steps and a moving bar. |
| `examples/map` | `Map` over a list under one bar. |
| `examples/many` | 256 `Map` items. The view calls `List(n)`. |
| `examples/tree` | Release, then fetch, compile, and package. |
| `examples/lines` | Three rows that rewrite until a newline. |
| `examples/rsync` | Parallel fake transfers. Each transfer rewrites one row. |
| `examples/triangle` | RGB triangle, one turn every four seconds. Plus and minus step the rate by 0.05. |
| `examples/perlin` | Animated Perlin noise. |
| `examples/compute` | Embedded shader `example.comp`. A path argument loads that shader. |
| `examples/scroll` | Rounded translucent boxes in a loop. |
| `examples/notepad` | An editor. The buffer stays in memory. |
| `examples/elm` | Two buttons that add and subtract an integer. |
| `examples/webview` | In-process page in the system web view. |
| `examples/spa` | Templ page as a single-page app. |
| `examples/music` | Drop a music directory, or pass one after `--`. Browse and play from an in-memory catalog. |
| `examples/tray` | Status item until Quit or interrupt. |
| `examples/sound` | No arguments lists sinks. Other arguments play those files. |
| `examples/filedialog` | Asks for files and prints each path. |
| `examples/welcome` | Picks a directory from the start screen. |

Arguments after `--` go to the program:

```bash
go run ./cmd/lewkit release run --config ./examples/compute/eletrocromo.json -- smoke
```

`smoke` prints a 4-byte probe. The probe needs a Vulkan compute device. A shader path loads that file.

`examples/music` takes an optional directory after `--`. No directory waits for a drop.

`examples/sound` with no arguments lists sinks. `mix OUT INPUTS...` writes one WAV. Other arguments play those files.
