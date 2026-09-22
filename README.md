# lewkit

lewkit is a reuse library of Go primitives. Another project imports one package and calls it. The module path is `github.com/lewtec/lewkit`. The module requires Go 1.27.

Rushed code is not sustainable long term and this is our tool of reuse.

`cmd/lewkit` is the program for demos, disassembly, and code generation. Type placement rules live in [SPEC.md](SPEC.md).

## Start here

Pick the package that owns the job. The full list is in [Code map](#code-map).

| Job | Start at |
| --- | --- |
| Name files inside a directory | `x/path` |
| Read or copy an archive | `x/fs` plus `x/fs/tar`, `x/fs/zip`, `x/fs/squashfs`, `x/fs/udf`, or `x/fs/wim` |
| Compress a byte stream | `x/compression` |
| Open a database and run queries | `x/db` |
| Run a task tree | `x/taskgroup` |
| Evaluate a tensor | `x/ndarray` |
| Open a window | `x/driver/window` |
| Disassemble machine code | `x/disasm` |
| Blit pixels on the CPU | `x/image` |
| Paint a window from a layout | `x/ui/gui` |

## Add the module

Add the `main` branch when you want the packages this file describes.

```bash
go get github.com/lewtec/lewkit@main
```

The `@latest` selector follows the highest version tag. That tag can lag `main`.

`dir` is an OS path. `path.New` is a slash name inside that directory. The function returns the bytes of `README.md`.

```go
import "github.com/lewtec/lewkit/x/path"

func readReadme(dir string) ([]byte, error) {
	root, err := path.Open(dir)
	if err != nil {
		return nil, err
	}
	return path.New("README.md").ReadFile(root)
}
```

Some packages register a factory in `init`. Blank-import the registry before the call.

```go
import (
	_ "github.com/lewtec/lewkit/x/compression/prelude"
	_ "github.com/lewtec/lewkit/x/db/postgres"
	_ "github.com/lewtec/lewkit/x/db/sqlite"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
)
```

`x/compression/prelude` registers gzip, brotli, lz4, zstd, xz, and bzip2. `x/driver/prelude` registers the window backends, the Vulkan compute facade, and the ndarray evaluators. The sqlite import registers `sqlite`, `sqlite3`, and `file`. A bare path and `:memory:` use that driver too.

The packages build with `CGO_ENABLED=0`.

## Run a command

Build the program from a clone.

```bash
go build -o lewkit ./cmd/lewkit
```

`lewkit --help` prints the command tree. `lewkit --version` prints the version from `x/release`.

Repeat `-v` to raise the log level. `--pprof` takes a directory or a listen address. `--sentry-dsn` sets the Sentry DSN. The same value can come from `SENTRY_DSN`.

### Commands

| Command | Result |
| --- | --- |
| `lewkit doctor` | Lists each driver interface and the implementation it selected. |
| `lewkit disasm hex HEX` | Prints instructions for a hex byte string. |
| `lewkit disasm raw PATH` | Prints instructions for a raw byte file. A `PATH` of `-` reads stdin. |
| `lewkit disasm file PATH` | Prints one text section of an ELF, PE, or Mach-O file. |
| `lewkit generate db DIR` | Writes sqlc packages and a shared Queries interface. |
| `lewkit generate prelude DIR` | Writes a blank-import prelude from each `root.go` under `DIR`. |
| `lewkit generate protobuf FILE` | Writes Go from a `.proto` file. |
| `lewkit completion` | Prints the bash `complete -C` line for this program. |
| `lewkit experiments demo NAME` | Runs one taskgroup demo in the terminal. |
| `lewkit experiments window NAME` | Runs one window demo. |

`lewkit disasm` shares `--architecture` (default `x86`), `--mode` (default `64`), `--syntax` (default `default`), `--address`, `--count`, and `--skip-data`. A `--count` of `0` prints every instruction. `lewkit disasm file` also takes `--section`.

`lewkit generate db` reads `sqlite/` and `postgres/` under `DIR`. `lewkit generate prelude` writes to stdout when you omit the output path. `lewkit generate protobuf` accepts `--package` when the file has no `go_package`.

### Taskgroup demos

These demos run in the terminal. A terminal is enough.

| Name | Result |
| --- | --- |
| `tasks` | Progress bars, logs, pools, and dependencies. |
| `plain` | The same schedule, with no progress view. |
| `nested` | An `Isolate` error boundary around child tasks. |
| `loop` | Five steps and a moving bar. |
| `map` | `Map` over a list under one bar. |
| `many` | 256 `Map` items. The view walks only `List(n)` rows. |
| `tree` | A deep tree: release, then fetch, compile, and package. |
| `lines` | Three rows that rewrite until a newline. |
| `rsync` | Parallel fake transfers. Each transfer rewrites one row. |

Run one by name.

```bash
lewkit experiments demo tasks
```

### Window demos

Run a window demo on a machine with a screen. The command opens a window.

| Name | Result |
| --- | --- |
| `triangle` | Draws the RGB triangle and turns it once per second. |
| `perlin` | Draws animated Perlin noise. |
| `compute` | Runs a compute shader in the window. |
| `scroll` | Draws rounded translucent boxes in a loop. |
| `notepad` | An editor that does not save the text. |
| `counter` | Two buttons that add and subtract an integer. |

`compute` runs `example.comp` when you omit the shader path. Pass a `.spv` or `.comp` path to run that shader. `--smoke` skips the window and prints a 4-byte probe. The probe needs a Vulkan compute device.

`triangle`, `perlin`, `scroll`, `notepad`, and `counter` accept `--width` and `--height`.

```bash
lewkit experiments window triangle
lewkit experiments window compute --smoke
```

## Code map

Each row is one package. Prefix the path with `github.com/lewtec/lewkit/`.

`x/ffi` and `x/ui` are directories with no Go package. An import of either path fails. [SPEC.md](SPEC.md) names `x/ui/tui` and `x/ui/web`. Those packages are absent until the first type lands.

### Program

| Path | Job |
| --- | --- |
| `cmd/lewkit` | The `lewkit` program. |
| `cmd/lewkit/experiments` | Demo commands. Only `cmd/lewkit` imports this package. |
| `report` | Sends an error to each registered reporter. |
| `report/sentry` | The Sentry reporter. |

### Files and bytes

| Path | Job |
| --- | --- |
| `x/path` | Slash names for an `io/fs` root. Call `path.Open` on the OS directory first. |
| `x/path/pick` | A predicate on a `path.Path`. |
| `x/fs` | Archive listings. `Walk` reads an `io/fs`. `Copy` writes a listing. `New` indexes one. |
| `x/fs/tar` | A tar archive as `io/fs`. |
| `x/fs/zip` | A ZIP archive as `io/fs`. |
| `x/fs/squashfs` | A SquashFS image as `io/fs`. |
| `x/fs/udf` | A UDF volume as `io/fs`. |
| `x/fs/wim` | One WIM image as `io/fs`. |
| `x/compression` | Stream codecs keyed by extension or magic bytes. |
| `x/compression/gzip` | The gzip codec. `brotli`, `lz4`, `zstd`, `xz`, and `bzip2` match this shape. |
| `x/compression/prelude` | Blank-import. Registers gzip, brotli, lz4, zstd, xz, and bzip2. |
| `x/db` | A database URL. `Open` migrates. `Value` runs sqlc queries. |
| `x/db/sqlite` | Registers `sqlite`, `sqlite3`, and `file`, plus a bare path and `:memory:`. |
| `x/db/postgres` | Registers the `postgres` scheme. |
| `x/db/generate` | Writes the Queries interface. `lewkit generate db` calls this package. |
| `x/io` | `Mkdirp` creates a directory and any missing parent. |
| `x/io/atomic` | Writes a file, or replaces a directory, then commits or rolls back. |
| `x/text` | Maps a UTF-8 byte offset to a line and a column. |

Blank-import `x/db/sqlite` or `x/db/postgres` before `db.Open`. Blank-import a codec, or `x/compression/prelude`, before `compression.Detect`.

### Tasks and process helpers

| Path | Job |
| --- | --- |
| `x/cmd` | Turns a Go struct into commands, flags, and bash completion. |
| `x/taskgroup` | Runs a task tree. A leaf takes IO, CPU, or internet from a pool. |
| `x/taskgroup/progress` | A bubbletea view of a taskgroup `Session`. |
| `x/thread` | Pins one goroutine to an OS thread. `thread.Run` starts that work from `main`. |
| `x/event` | A publish and subscribe bus. `CreateTimer` is a tick channel for a context. |
| `x/future` | A value you read with `Get`. `Peek` fails until that value is ready. |
| `x/singleton` | Runs one init function a single time. |
| `x/profile` | Records pprof data in a directory or serves it on HTTP. |
| `x/release` | The version string. `lewkit --version` prints it. |
| `x/test` | Test helpers for globals, closers, iterators, and readers. |
| `x/auth` | `HashPassword` hashes a password with bcrypt. |
| `x/generate` | Helpers shared by the generator packages. |
| `x/generate/prelude` | Writes a blank-import file from `root.go` markers. |
| `x/generate/protobuf` | Writes Go from one `.proto` file. |

`x/event` also exposes `FPS`, a smoothed frame rate.

### Numbers and pictures

| Path | Job |
| --- | --- |
| `x/ndarray` | N-dimensional views and one fused kernel of 21 ALU ops. |
| `x/ndarray/nn` | Convolution, pool, matmul, and a linear layer. |
| `x/ndarray/onnx` | Reads an ONNX model and lowers the graph onto ndarray. |
| `x/ndarray/image` | `Fill` builds an `(h, w, 4)` color. `Eval` packs it into `image.RGBA`. |
| `x/image` | CPU blit, text, and the RGB triangle (`Triangle`, `TriangleTurn`). |
| `x/graph` | A directed graph. It prints DOT or Mermaid. |
| `x/ui/gui` | A bubbletea-shaped loop. `View` returns a layout node. |

`Tensor` holds a lazy op tree. `Eval` runs that tree. `Open` picks an evaluator from `x/driver/ndeval`.

`x/ui/gui` layout is `Box`, `Flex`, and `Stack` on the CPU. One kernel paints rounded rects and glyph ink. `Run` paints a window the caller opened. `Open` opens the window, then calls `Run`.

### Host and foreign code

| Path | Job |
| --- | --- |
| `x/driver` | A registry of capabilities. `List` enumerates. `Get` opens the first match. |
| `x/driver/prelude` | Blank-import. Registers the drivers in this table. |
| `x/driver/window` | A resizable window. The back buffer is an `image.RGBA`. |
| `x/driver/window/cocoa` | The macOS window backend. |
| `x/driver/window/win32` | The Windows window backend. |
| `x/driver/window/x11` | The X11 window backend. |
| `x/driver/window/mem` | An in-memory window for tests. |
| `x/driver/vulkan` | Compute GPUs as drivers. `Open` and `List`. This package is a facade. |
| `x/driver/ndeval` | Evaluator factories. The CPU factory registers at init. Vulkan wraps the selected GPU. |
| `x/disasm` | Decodes machine code. This package is a facade over Capstone. |
| `x/ffi/native` | Loads a shared library and binds functions. No cgo. |
| `x/ffi/native/vulkan` | The libvulkan binding. `Shader` takes SPIR-V. |
| `x/ffi/wasm` | Loads embedded WebAssembly with wazero. |
| `x/ffi/wasm/capstone` | The Capstone binding. |
| `x/ffi/wasm/glsl` | Compiles Vulkan GLSL to SPIR-V. |

`window.Open` returns the window. `Frame` is the back buffer. `Draw` swaps it. `Animate` calls a paint func on a tick. `Subscribe` yields `Resize`, `Expose`, `Close`, `Pointer`, `Scroll`, and `Key`.

On macOS, `window.Open` must run on the process main thread. Call `thread.Run` from `main` so the call stays there. The `lewkit` program already does this.

Blank-import `x/driver/prelude`, or one window backend, before `window.Open`. Blank-import the prelude or `x/driver/ndeval` before `ndarray.Open`.

## Choose an import

Use one term for each concept.

| Concept | Term in this file | Banned |
| --- | --- | --- |
| A reusable type in this module | primitive | toolkit, widget |
| The package that loads a C library | binding | facade |
| The package callers use instead of that binding | facade | binding |
| `cmd/lewkit/experiments` | demo | library |

Import the facade when one exists.

| You want | Import |
| --- | --- |
| Disassembly | `x/disasm` |
| A GPU, a buffer, or a command buffer | `x/driver/vulkan` |
| An evaluator for `Tensor.Eval` | `x/ndarray` (`Open`) |
| The libvulkan loader | `x/ffi/native/vulkan` |
| GLSL compiled to SPIR-V | `x/ffi/wasm/glsl` |

`x/driver/vulkan` keeps the libvulkan device private. It re-exports `Buffer`, `Shader`, and `Cmd`. `x/driver/ndeval` compiles a kernel with `x/ffi/wasm/glsl` and runs that kernel on the Vulkan facade. `x/disasm` calls the Capstone binding.

## Develop this repository

From a clone, test and build with the Go tool.

```bash
go test ./...
go build -o lewkit ./cmd/lewkit
```

`mise.toml` defines that test task, plus `lint`, `format`, and `codegen`. `codegen` runs `go generate ./...`. `lint` and `format` call workspaced.

Read [SPEC.md](SPEC.md) before you add a type. A new type has one owner package.

## License

The license text is [LICENSE.md](LICENSE.md). Commercial use needs a separate agreement. Write to lucas@lew.tec.br.
