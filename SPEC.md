# lewkit Specification

lewkit is a reuse library of primitives. This constitution assigns each type one owner package. Bubbletea types, templ templates, and pixel-frame transformers live under `x/ui`. Host, engine, and Session viewers stay in their packages. This file is the only SPEC.md.

Status: approved
Genre: library

The key words MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY in this
document are to be interpreted as described in BCP 14 (RFC 2119,
RFC 8174) when, and only when, they appear in all capitals.

## Intention

Job: When an author adds a type, this document names the one package that owns it.

Non-goals:

1. A widget toolkit.
2. A Material catalog.
3. A Flutter widget tree (Column, Element, RenderObject).
4. Treating `x/ui` as an importable UI library.
5. The ndarray ISA, `Tensor`, and `Evaluator`.
6. Window `Open`, `Frame`, `Fit`, `Present`, and `Animate`.
7. A `Widget` type shared by `tui`, `web`, and `gui`.
8. Moving `x/taskgroup/progress`.
9. Moving triangle, perlin, and compute demos.
10. A second constitution at any other path.

Inherited C (cite the file):

- README: reuse library of primitives, not an application.
- `go.mod`: Go 1.27, module `github.com/lewtec/lewkit`, `charm.land/bubbletea/v2`.
- `path:x/taskgroup/progress`: bubbletea viewer of a taskgroup `Session`.
- `path:x/ndarray`: `Tensor`, 21 ALU ops, `Evaluator`.
- `path:x/driver/window`: host window, `Fit`, `Present`, `Animate`.
- `path:x/ndarray/image`: pack `(h,w,4)` into `image.RGBA`.
- `path:x/image`: CPU blit and `Label`.
- `path:x/image/convert`: PNG, JPEG, ICO, and ICNS icon bytes.
- `path:cmd/lewkit/experiments`: triangle, perlin, compute demos.
- templ is the web toolkit. A tag for one registered asset lives in that asset package. Page templates live in `x/ui/web`.

## Technique

| ID | Input | Rule | Output |
|----|-------|------|--------|
| TEC-01 | A new type | Classify by native export. The caller-imported type picks the owner in the placement table. A type that uses a toolkit to view another package's type stays next to that type. | Exactly one owner package |
| TEC-02 | A type that matches two of `tui`, `web`, `gui` | Split it into two types. Each type has one native export. | No shared `Widget` in `x/ui` |
| TEC-03 | A type under `x/ui` | The type is a component. The application opens the host. `gui.Run` drives a supplied `Window` the way `tea.Program` drives a terminal. | `window.Open` stays outside `gui` |
| TEC-04 | An import edge | `tui`, `web`, and `gui` MAY import host and engine. Host and engine MUST NOT import them. | Acyclic ownership |
| TEC-05 | A C library loaded with `dlopen` | Place the package under `x/ffi/native`. The package imports `x/ffi/native`. | One binding at `x/ffi/native/<name>` |
| TEC-06 | A C library loaded with the wasm runtime | Place the package under `x/ffi/wasm`. The package imports `x/ffi/wasm`. | One binding at `x/ffi/wasm/<name>` |
| TEC-07 | A driver over a binding | The driver imports the binding. The driver does not import `x/ffi/native`. The driver does not import `x/ffi/wasm`. | A facade |

## Tooling

| TEC | Tool | Relation | We do not | Cite |
|-----|------|----------|-----------|------|
| TEC-01 | this SPEC | implement | add a fourth package under `x/ui` | none |
| TEC-02 | this SPEC | implement | unify `tui`, `web`, and `gui` behind one interface | none |
| TEC-03 | existing host loops | wrap | start the host loop from `x/ui` packages | path:x/driver/window path:x/taskgroup/progress |
| TEC-04 | ndarray + window | wrap | relocate Present into `gui`; relocate Tensor into `gui` | path:x/ndarray path:x/driver/window |
| TEC-05 | purego | wrap | call `Dlopen` from a binding | go.mod path:x/ffi/native |
| TEC-06 | wazero | wrap | call wazero from a binding | go.mod path:x/ffi/wasm |
| TEC-07 | this SPEC | implement | return the binding device from `x/driver/vulkan` | path:x/driver/vulkan path:x/ffi/native/vulkan |
| TEC-01 tui | bubbletea v2 | adopt | write a terminal runtime | go.mod path:x/taskgroup/progress |
| TEC-01 gui | ndarray 21-op graph | wrap | read the picture back to the host before present | path:x/ndarray |
| TEC-01 web | templ | adopt | write an HTML runtime | go.mod path:x/ui/web |

| Cell | Pick | C or D | Implements | Cite if C |
|------|------|--------|------------|-----------|
| Language | Go 1.27 | C | all | go.mod |
| Runtime | the calling Go process | C | TEC-03 | go.mod |
| Persistence | none | C | this SPEC stores no data | none |
| UI | not a UI library; three component packages under `x/ui` | D | TEC-01 | |
| Packaging | this Go module | C | all | go.mod |
| Identity | none | C | packages have no user identity | none |
| Host OS | window backends already in tree | C | TEC-03 TEC-04 | path:x/driver/window |

## Terminology

| Concept | Approved | Banned |
|---------|----------|--------|
| primitive | a reusable type in this module | UI library, widget toolkit, Flutter |
| namespace | `x/ui` as a directory of three packages | parent `Widget`; importable `x/ui` API |
| component | bubbletea type; templ template; tensor transformer | widget |
| transformer | `Picture.Render` of a `Node` tree: a fused `(h,w,4)` `Tensor` headless, and the same fills drawn on a swapchain | painter, widget, GUI framework |
| Model | Init, Update, View (bubbletea shape; View is a layout Node) | Widget, Flutter Element |
| host | `x/driver/window` | gui, windowing toolkit |
| engine | `x/ndarray` | tinygrad |
| binding | C library package nested under the loader package it imports | facade |
| facade | package that imports a binding and does not import `x/ffi/native`. It does not import `x/ffi/wasm` | binding |
| viewer | code that uses a toolkit to show another package's type | component package |
| demo | `cmd/lewkit/experiments` | component package |

`tui`, `web`, and `gui` are package names. They are not a product.

## Types

| Type | Exported | Identity or value | Mutable | Nil/error | Callers MUST NOT |
|------|----------|-------------------|---------|-----------|------------------|
| `x/ui` | no Go API | names `tui`, `web`, `gui` | MUST NOT grow types | the directory MAY have no Go package | import `x/ui` |
| `x/ui/tui` | bubbletea types for callers to compose | value | catalog MAY grow | package MAY be absent until the first type | put Session viewers here |
| `x/ui/web` | page templ templates | value | catalog MAY grow | package MAY be absent until the first page template | put a page template outside `web`; put an asset tag outside its asset package |
| `x/ui/gui` | `Model`, `Msg`, `Cmd`, `Run`; `View` is a layout `Node` | value | catalog MAY grow | package MAY be absent until the first transformer | own the host; own the engine; call `window.Open` |
| `x/driver/window` | `Open`, `Frame`, `Fit`, `Present`, `Animate` | host identity is the opened window | protocol stays here | missing driver is the existing window error | move Present into `gui` |
| `x/driver/tray` | `Open`, `Tray`, `Icon`, `Item` | host status item | protocol stays here | missing session bus or host is the tray error | import `x/ffi/wasm` |
| `x/driver/colorscheme` | `Current`, `Watch`, `Scheme` | system color scheme | protocol stays here | missing portal or host is `driver.ErrUnavailable` | import `x/ui/gui`; import `x/driver/webview`; import `x/ffi/wasm` |
| `x/ndarray` | `Tensor`, ops, `Evaluator` | engine | ISA stays here | existing ndarray errors | import `x/ui/gui` |
| `x/ndarray/image` | pack `(h,w,4)` into `image.RGBA` | value | packing stays here | existing pack errors | hold bubbletea types; hold templ; hold transformers |
| `x/image` | CPU blit, `Label` | value | blit stays here | existing blit errors | hold bubbletea types; hold templ; hold transformers |
| `x/image/convert` | `Decode`, `Square`, `EncodePNG`, `EncodeICO`, `EncodeICNS`, `ARGB` | value | icon bytes stay here | bad bytes are `ErrFormat` | import `x/driver` |
| `x/sound` | `Format`, `Mixer`, `Mix`, `Pipeline`, `Decode`, `Register`, `WriteWAV`, `ReadWAV` | PCM value | mixing, seek, and the decoder registry stay here | `ErrFormat`, `ErrFrame`, `ErrClosed`, `ErrSeek` | open a host device; import `x/driver`; import `x/sound/mp3`; import `x/sound/ogg` |
| `x/sound/mp3` | MP3 `Decoder` | registered decoder | decode stays here | mp3 decode error | import `x/driver` |
| `x/sound/ogg` | Ogg Vorbis `Decoder` | registered decoder | decode stays here | vorbis decode error | import `x/driver` |
| `x/sound/prelude` | blank import of the decoders | registry side effect | imports stay here | duplicate `Register` panics | decode by itself |
| `x/driver/audio_play` | `Open`, `Sinks`, `Config` | host sink is the playback writer | protocol stays here | missing driver is `driver.ErrUnavailable` | import `x/ffi/native`; import `x/ffi/wasm` |
| `x/ffi/native/pulse` | `Playback`, `List`, `Stream` | libpulse binding | simple playback stays here | pulse error text | import `x/ffi/wasm`; import `x/driver` |
| `x/driver/audio_play/pulse` | PulseAudio `Open` | facade of the pulse binding | selection stays here | missing library is `driver.ErrIncompatible` | import `x/ffi/native`; return `pa_simple` |
| `x/driver/audio_play/mem` | in-memory `Open` | test sink | capture stays here | incompatible unless `LEWKIT_AUDIO_PLAY_MEM` is set | play on a host device |
| `x/ffi/native/winmm` | `Open`, `Devices`, `Stream` | winmm binding | waveOut playback stays here | waveOut error text | import `x/ffi/wasm`; import `x/driver` |
| `x/driver/audio_play/winmm` | waveOut `Open` | facade of the winmm binding | selection stays here | missing library is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/ffi/native/coreaudio` | `Open`, `Devices`, `Stream` | AudioQueue binding | playback stays here | CoreAudio error text | import `x/ffi/wasm`; import `x/driver` |
| `x/driver/audio_play/coreaudio` | AudioQueue `Open` | facade of the coreaudio binding | selection stays here | missing framework is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/filedialog` | `Choose`, `Request`, `Filter` | host file or folder dialog | protocol stays here | `ErrCanceled`, `ErrRequest`; missing driver is `driver.ErrUnavailable` | import `x/ffi/native`; import `x/ffi/wasm` |
| `x/driver/filedialog/portal` | `Choose`, `Available` | portal file chooser call | GTK and KDE service names stay here | missing bus name is `driver.ErrIncompatible` | import `x/ffi/native`; show a dialog by itself |
| `x/driver/filedialog/gtk` | GTK `Choose` | facade of the gtk portal backend | selection stays here | missing gtk portal is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/filedialog/qt` | Qt `Choose` | facade of the KDE portal backend | selection stays here | missing KDE portal is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/filedialog/cocoa` | `NSOpenPanel` and `NSSavePanel` | macOS file dialog | selection stays here | missing main thread is the file dialog error | import `x/ffi/wasm` |
| `x/driver/filedialog/win32` | common item dialog `Choose` | Windows file dialog | selection stays here | missing main thread is the file dialog error | import `x/ffi/wasm` |
| `x/taskgroup/progress` | bubbletea viewer of `Session` | viewer of `Session` | stays next to `Session` | existing TUI skip rules | move into `x/ui/tui` |
| `x/ffi` | no Go API | names `native`, `wasm` | MUST NOT grow a Go package | directory has no `.go` file | import `x/ffi` |
| `x/ffi/native` | `Open`, `Func`, `Symbol`, `Register` | direct C ABI | loader stays here | purego error | import `x/ffi/native/vulkan`; import `x/ffi/native/pulse`; import `x/ffi/native/winmm`; import `x/ffi/native/coreaudio` |
| `x/ffi/wasm` | `Compile`, `Instance` | wasm runtime | host stays here | existing wasm errors | import `x/ffi/wasm/glsl`; import `x/ffi/wasm/capstone` |
| `x/ffi/native/vulkan` | `Device`, `Buffer`, `Shader`, `Cmd`, swapchain `Draw` | libvulkan binding | compute plus one graphics draw for the swapchain | existing vulkan errors | import `x/ffi/wasm` |
| `x/ffi/wasm/glsl` | `Compile`, `Load`, `IsSPIRV` | glslang binding | compiler stays here | existing glsl errors | import `x/ffi/native` |
| `x/ffi/wasm/capstone` | `Open`, `Handle`, `Instruction` | Capstone binding | guest stays here | capstone error text | import `x/ffi/native` |
| `x/ffi/native/webkitgtk` | `Load`, `Symbols` | WebKitGTK 6 and GTK 4, dlopen | loader stays here | missing library is `ErrUnavailable` | import `x/ffi/wasm`; listen on a port |
| `x/ffi/native/webkit` | `Load` | WebKit.framework, dlopen | loader stays here | missing framework is `ErrUnavailable` | import `x/ffi/wasm`; listen on a port |
| `x/ffi/native/webview2` | `Available`, `CreateEnvironment` | WebView2Loader.dll | loader stays here | missing loader is `ErrUnavailable` | import `x/ffi/wasm`; listen on a port |
| `x/driver/webview` | `Open`, `View` | OS web view; page bytes and script messages stay in-process | protocol stays here | missing driver is the existing driver error | `net.Listen`; launch a browser; import `x/ffi/native` |
| `x/driver/vulkan` | `Open`, `List`, `Device` with `Buffer`, `Compile`, `Begin` | facade of the vulkan binding | selection stays here | existing vulkan errors | return the binding `Device`; import `x/ffi/native`; import `x/ffi/wasm` |
| `x/disasm` | `Engine`, object files, hex | facade of capstone | formats stay here | existing disasm errors | import `x/ffi/wasm` |
| `x/driver/ndeval` | CPU and Vulkan `Evaluator` factories | facade | factories stay here | existing ndarray errors | import `x/ffi/native/vulkan`; import `x/ffi/wasm` |
| `cmd/lewkit/experiments` | commands, not a library | demo | demos MAY stay | command failure | import experiments as a component package |

## Invariants

| ID | Predicate | On | Forbidden bypass |
|----|-----------|----|------------------|
| INV-01 | A type has exactly one owner in the placement table | every new type | a fourth package under `x/ui`; a type copied into two owners |
| INV-02 | `x/ui` exports no types | `x/ui` | `Widget`, shared `Color`, shared `Align` |
| INV-03 | `tui` native export is a bubbletea type | `x/ui/tui` | templ files; tensor transformers |
| INV-04 | `web` native export is a page templ template. A tag for one registered asset lives in that asset package | `x/ui/web`; `x/http/asset` | bubbletea types; tensor transformers; an asset tag in `web` |
| INV-05 | `gui.Model.View` returns a layout `Node`; headless `Run` paints a `(h,w,4)` tensor; a swapchain paints recorded fills, and a mounted tensor is that picture's kernel | `x/ui/gui` | `View` returning a tensor; `window.Present` declared here; a host readback of the picture before present |
| INV-06 | A viewer stays next to the type it shows | `x/taskgroup/progress` | move progress into `x/ui/tui` because it uses bubbletea |
| INV-07 | `tui`, `web`, and `gui` MUST NOT import each other | those packages | `gui` emitting HTML; `tui` importing `gui` |
| INV-08 | Host and engine MUST NOT import `tui`, `web`, and `gui` | `x/driver/window`, `x/ndarray` | `window` depending on `gui` |
| INV-09 | This repository has one constitution: `SPEC.md` at the repo root | this file | `x/ui/SPEC.md`; a second SPEC beside this file |
| INV-10 | `gui.Run` consumes a caller-supplied `Window`. It MUST NOT call `window.Open`. | `x/ui/gui` | `gui` opening a host window |
| INV-11 | This module is not a UI library | this repository | advertising `x/ui` as the product; a Flutter widget tree as the public API |
| INV-12 | Host window events include `Resize`, `Expose`, `Close`, `Pointer`, `Scroll`, and `Key` | `x/driver/window` | pointer `Msg` types that the host does not emit |
| INV-13 | `x/ffi` has no Go package | `x/ffi` | a `.go` file whose package is `ffi` |
| INV-14 | `x/ffi/native` does not import a nested binding | `x/ffi/native` | an import of `vulkan`, `pulse`, `winmm`, `coreaudio`, `webkitgtk`, `webkit`, or `webview2` |
| INV-15 | `x/ffi/wasm` does not import `x/ffi/wasm/glsl` | `x/ffi/wasm` | that import |
| INV-16 | `x/ffi/wasm` does not import `x/ffi/wasm/capstone` | `x/ffi/wasm` | that import |
| INV-17 | `x/ffi/native/vulkan` imports `x/ffi/native` | that package | an import of `x/ffi/wasm` |
| INV-18 | `x/ffi/wasm/glsl` imports `x/ffi/wasm` | that package | an import of `x/ffi/native` |
| INV-19 | `x/ffi/wasm/capstone` imports `x/ffi/wasm` | that package | an import of `x/ffi/native` |
| INV-20 | `x/driver/vulkan.Device` does not return the binding device | `x/driver/vulkan` | a method whose result type is the binding `Device` |
| INV-21 | `x/driver/vulkan` does not import `x/ffi/native` | `x/driver/vulkan` | that import |
| INV-22 | `x/driver/ndeval` does not import `x/ffi/native/vulkan` | `x/driver/ndeval` | that import |
| INV-23 | `x/driver/ndeval` does not import `x/ffi/wasm` | `x/driver/ndeval` | that import |
| INV-24 | `x/disasm` does not import `x/ffi/wasm` | `x/disasm` | that import |
| INV-25 | `id_unix.go` loads libc through `x/ffi/native` | `x/thread/id_unix.go` | an import of `x/ffi/wasm` |
| INV-26 | `x/driver/window/cocoa` loads frameworks through `x/ffi/native` | cocoa darwin files | an import of `x/ffi/wasm` |
| INV-27 | `main_darwin.go` loads `pthread_main_np` through `x/ffi/native` | `x/thread/main_darwin.go` | an import of `x/ffi/wasm` |
| INV-28 | `x/ffi/native/webkitgtk` imports `x/ffi/native` | that package | an import of `x/ffi/wasm` |
| INV-29 | `x/driver/webview` does not import `x/ffi/native` | `x/driver/webview` | that import |
| INV-30 | `Open` does not listen on a socket. The page is memory or `fs.FS`. Script messages are the Go bridge | `x/driver/webview` | `net.Listen`; a loopback URL |
| INV-31 | `x/ffi/native/webkit` imports `x/ffi/native` | that package | an import of `x/ffi/wasm` |
| INV-32 | `x/driver/tray` does not import `x/ffi/wasm` | `x/driver/tray` | that import |
| INV-33 | `x/sound` does not import `x/driver`, `x/sound/mp3`, or `x/sound/ogg` | `x/sound` | that import |
| INV-34 | `x/driver/audio_play` does not import `x/ffi/native` | `x/driver/audio_play` | that import |
| INV-35 | `x/driver/audio_play/pulse` imports `x/ffi/native/pulse` and does not import `x/ffi/native` | `x/driver/audio_play/pulse` | an import of `x/ffi/native` |
| INV-36 | `x/ffi/native/pulse` imports `x/ffi/native` | that package | an import of `x/ffi/wasm` or `x/driver` |
| INV-37 | `x/ffi/native/winmm` imports `x/ffi/native` | that package | an import of `x/ffi/wasm` or `x/driver` |
| INV-38 | `x/ffi/native/coreaudio` imports `x/ffi/native` | that package | an import of `x/ffi/wasm` or `x/driver` |
| INV-39 | `x/driver/audio_play/winmm` imports `x/ffi/native/winmm` and does not import `x/ffi/native` | `x/driver/audio_play/winmm` | an import of `x/ffi/native` |
| INV-40 | `x/driver/audio_play/coreaudio` imports `x/ffi/native/coreaudio` and does not import `x/ffi/native` | `x/driver/audio_play/coreaudio` | an import of `x/ffi/native` |
| INV-41 | `x/driver/filedialog` does not import `x/ffi/native` or `x/ffi/wasm` | `x/driver/filedialog` | that import |
| INV-42 | `x/driver/filedialog/gtk` does not import `x/ffi/native` | `x/driver/filedialog/gtk` | that import |
| INV-43 | `x/driver/filedialog/qt` does not import `x/ffi/native` | `x/driver/filedialog/qt` | that import |
| INV-44 | `x/driver/filedialog/cocoa` does not import `x/ffi/wasm` | `x/driver/filedialog/cocoa` | that import |
| INV-45 | `x/driver/filedialog/win32` does not import `x/ffi/wasm` | `x/driver/filedialog/win32` | that import |
| INV-46 | `x/driver/colorscheme` does not import `x/ui/gui`, `x/driver/webview`, or `x/ffi/wasm` | `x/driver/colorscheme` | that import |

## Errors

| Public operation | Bad input | One reaction |
|------------------|-----------|--------------|
| Place a type | Matches two of `tui`, `web`, `gui` | Split into two types. MUST NOT add a `Widget` in `x/ui`. |
| Place a type | Matches no row in the table | Leave it in its existing owner. MUST NOT add a fourth package under `x/ui`. |
| Place a type | Uses bubbletea to view `Session` | Keep it in `x/taskgroup/progress`. |
| Place a type | First page templ template in the module | Create `x/ui/web`. MUST NOT put the file in `gui`. MUST NOT put the file in `tui`. |
| Place a type | Tag for one registered browser asset | Put the templ file in that asset package. |
| Place a type | First reusable tensor transformer for a pixel frame | Create `x/ui/gui`. MUST NOT leave it in `x/ndarray`. MUST NOT leave it in `x/driver/window`. |
| Place a C library | The package calls `native.Open` and its parent is not `x/ffi/native` | Move the package under `x/ffi/native`. |
| Place a C library | The package calls `wasm.Compile` and its parent is not `x/ffi/wasm` | Move the package under `x/ffi/wasm`. |
| Export from `x/ui` | A Go type on the namespace | Move the type into the one of `tui`, `web`, `gui` that needs it. |
| `gui.Run` | nil `Model` | Return `ErrModel`. |
| `gui.Run` | nil `Window` | Return `window.ErrClosed`. |
| `gui.Model.View` | nil `Node` | Return `ErrView`. Do not `Draw`. |
| `filedialog.Choose` | `Save` with `Folder` or `Multiple` | Return `ErrRequest`. |

## Actors

N/A: genre=library

## Quality

| Concern | Measure, or why it cannot happen |
|---------|----------------------------------|
| compatibility | A row's import path in the types table is stable. A rename is a change to this SPEC. |
| error model | Dual-home types are split at review. There is no runtime for Place. |

## Security

In scope: none. This SPEC places source files.

Why it cannot happen: placement does not handle untrusted input. It does not handle credentials. It does not handle a network boundary.

Residual risk: a later component catalog MUST add its own security row if it grows a surface that does.

## Success

- [ ] A reusable bubbletea type is imported from `x/ui/tui`.
- [ ] `window.Present` is not declared in `x/ui/gui`.
- [ ] `x/taskgroup/progress` still views `Session` with bubbletea.
- [ ] `x/ndarray` does not import `x/ui/gui`.
- [ ] The only `SPEC.md` in this repository is this file.
- [ ] `x/ui` has no exported Go type.
- [ ] README still describes lewkit as a reuse library of primitives.
- [ ] `gui.Model` is Init, Update, View. View is a layout Node, not a tensor.
- [ ] `x/ffi` contains no `.go` file.
- [ ] `x/driver/vulkan` does not declare `Native`.
- [ ] `x/driver/ndeval` does not import `x/ffi/native/vulkan`.
- [ ] `x/disasm` does not import `x/ffi/wasm`.
- [ ] `x/ffi/native/vulkan` does not import `x/ffi/wasm`.

## Later work

1. Reusable bubbletea types in `x/ui/tui`.
2. Text input (IME) and mapped key names.
3. Extract triangle and perlin from experiments into `gui` only after they are reusable transformers.

## Assumptions

| ID | Fact | If false |
|----|------|----------|
| AS-01 | bubbletea v2 remains the tui toolkit | Change the tui tooling row. Do not invent a terminal runtime. |
| AS-02 | templ remains the intended web toolkit | Change the web later-work item before adding `x/ui/web`. |
| AS-03 | Pointer coordinates are client pixels with the same origin as `Frame` | Change the cocoa Y flip / scale if a host uses another origin. |

## Decision history

- 2026-09-20 grill: native export placement; SPEC at repo root. Rejected: nested `x/ui/SPEC.md`; a shared `Widget`; moving progress into `tui`; moving triangle into `gui` now; calling this a UI library.
- 2026-09-20: `gui` follows bubbletea (`Model` / `Msg` / `Cmd` / `Run`) with tensors in `View`. Host events are `Resize`, `Expose`, `Close` only. Rejected: Flutter widget tree as the public API; `gui` calling `window.Open`.
- 2026-09-21: `gui.Model.View` returns a layout `Node`. `Run` paints it through `Picture` to a `(h,w,4)` tensor.
- 2026-09-20: window bus adds `Pointer`, `Scroll`, and `Key`. `gui.Run` forwards them. Marquee drag/wheel/space.
- 2026-09-21: C libraries live under the mechanism that loads them. `x/ffi/native/vulkan`, `x/ffi/wasm/glsl`, `x/ffi/wasm/capstone`. `x/driver/vulkan`, `x/driver/ndeval`, and `x/disasm` are facades. `x/ffi` is not a Go package. `x/thread` and cocoa call `x/ffi/native`.
- 2026-09-23: templ is adopted. A tag for one registered asset lives in that asset package. Page templates stay in `x/ui/web`. htmx, tailwindcss, jquery, and sakuracss are blank-import assets served from `/__lewkit__/`. The first page template is `x/ui/web` `Page`.
- 2026-09-24: light or dark is `x/driver/colorscheme`. Web views push it into the page without a reload. `gui.Run` delivers `SchemeMsg`.
