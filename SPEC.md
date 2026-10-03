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
9. Moving triangle, perlin, and compute demos into `x/ui/gui`.
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
- `path:examples`: triangle, perlin, compute demos.
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
| TEC-08 | Java called from Go | Place the call in `x/ffi/jni`. The package imports `x/ffi/native`. Callers name a class and pass Go values. Field reads stay there. An interface callback uses the one `lewkit.GoProxy` dispatcher. | One binding at `x/ffi/jni` |

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
| TEC-08 | JNI | wrap | add a JNI export per Java call | path:x/ffi/jni |
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
| demo | `examples/<name>` | component package |

`tui`, `web`, and `gui` are package names. They are not a product.

## Types

| Type | Exported | Identity or value | Mutable | Nil/error | Callers MUST NOT |
|------|----------|-------------------|---------|-----------|------------------|
| `x/ui` | no Go API | names `tui`, `web`, `gui` | MUST NOT grow types | the directory MAY have no Go package | import `x/ui` |
| `x/ui/tui` | bubbletea types for callers to compose | value | catalog MAY grow | package MAY be absent until the first type | put Session viewers here |
| `x/ui/web` | page templ templates | value | catalog MAY grow | package MAY be absent until the first page template | put a page template outside `web`; put an asset tag outside its asset package |
| `x/ui/gui` | `Model`, `Msg`, `Cmd`, `Run`, `Welcome`, `EnsureDir`, `Pick`; `View` is a layout `Node` | value | catalog MAY grow | package MAY be absent until the first transformer; empty dir with no terminal is `ErrNeedWindow` | own the host; own the engine; call `window.Open`; declare `Open` |
| `x/driver/window` | `Open`, `Frame`, `Fit`, `Present`, `Animate` | host identity is the opened window | protocol stays here | missing driver is the existing window error | move Present into `gui` |
| `x/driver/tray` | `Open`, `Tray`, `Icon`, `Item` | host status item | protocol stays here | missing session bus or host is the tray error | import `x/ffi/wasm` |
| `x/driver/daynight` | `Current`, `Watch`, `Mode` | light or dark | protocol stays here | missing portal or host is `driver.ErrUnavailable` | import `x/ui/gui`; import `x/driver/webview`; import `x/ffi/wasm` |
| `x/driver/daynight/android` | Android `Current`, `Watch` | night bit from the binder activity configuration | the binder read stays here | no binder is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/notification` | `Notify`, `Notification` | one local alert | protocol stays here | missing backend is `driver.ErrUnavailable` | import `x/ffi` |
| `x/driver/clipboard` | `WriteText`, `WriteImage` | host clipboard | protocol stays here | missing tool is `driver.ErrIncompatible` | import `x/ffi` |
| `x/driver/clipboard/android` | Android `WriteText`, `WriteImage` | clipboard text | text stays here | no Java VM is `driver.ErrIncompatible`; an image is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/opener` | `Open` | launch a file or URL | protocol stays here | missing opener is `driver.ErrUnavailable` | import `x/ffi` |
| `x/driver/dirs` | `Resolve`, `Dirs` | per-app data, cache, config, inbox | protocol stays here | bad app id is `ErrInvalidAppID` | hardcode a product name in the path |
| `x/driver/dirs/android` | Android `Resolve` | files, cache, and config from the package data dir | paths stay here | not android or no binder is `driver.ErrIncompatible` | import `x/ffi/native`; import `x/driver/webview` |
| `x/driver/bundle` | `Resolve`, `Root`, `SharePath` | stamped reverse-domain tree plus web profile | protocol stays here | missing id is `release.ErrAppIDRequired` | import `x/driver/webview`; replace `x/driver/dirs` |
| `x/driver/thread` | `Driver` | UI thread for this process | protocol stays here | JNI without a Java looper is `driver.ErrIncompatible` | import `x/ui/gui` |
| `x/release` | `Version`, `AppID`, `ValidateAppID`, `Name` | the runtime stamp and the short product name | `version`, the reverse-domain id, and `name` stay here | empty id is `ErrAppIDRequired`; an invalid name keeps the built-in default | import `x/driver`; a second version `-X`; a path, title, or protocol name that writes the product literal instead of calling `Name` |
| `x/entry` | `Main`, `MainFrom`, `Run`, `After` | process startup for apps and commands | the signal context, UI thread, and taskgroup session stay here | a second progress view is skipped | import `x/ui/gui`; start the session before command flags are parsed |
| `x/app` | `Web`, `GUI`, `Open`, `Run` | windows of one process; each is a web handler or a GUI model; a GUI model is `window.Open` then `gui.Run`; Run returns when the last window closes | the session stays here | invalid id is `release.ErrAppIDNotReverseDNS`; a loopback host with a GUI model fails | call `window.Open` beside `App.Run`; call `gui.Open` |
| `x/build` | `Desktop`, `Android`, `Mac`, `IOS` | archives and packaged hosts | packaging stays here; `x/build/version.Info` is packaging metadata | existing build errors | import `x/driver/webview`; `-X` `x/build/version.Version` |
| `x/driver/share` | `Out`, `Item` | text, URL, or files to another app | protocol stays here | empty item is `ErrEmptyItem` | the eletrocromo JSONL host file |
| `x/driver/volume` | `SetVolume`, `GetVolume`, `ToggleMute`, `Increase`, `Decrease`, `StatusNotification` | sink volume 0..1 | protocol stays here | missing pactl is `driver.ErrIncompatible` | play PCM; import `x/driver/audio_play`; post the alert here |
| `x/driver/volume/android` | Android `SetVolume`, `GetVolume`, `ToggleMute` | music stream volume 0..1 | the binder read stays here | no binder is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/brightness` | `SetBrightness`, `Status`, `Increase`, `Decrease`, `StatusNotification` | display brightness | protocol stays here | missing brightnessctl is `driver.ErrIncompatible` | import `x/ffi`; post the alert here |
| `x/driver/brightness/android` | Android `SetBrightness`, `Status` | foreground window override | the override stays here | no Java VM is `driver.ErrIncompatible` | import `x/ffi/native`; `DisplayManager.setBrightness` |
| `x/driver/battery` | `BatteryStatus`, `BatteryLevel` | charging state; level 0..100 | protocol stays here | no battery is `ErrNoBattery`; no level is `ErrUnknownLevel` | import `x/ffi` |
| `x/driver/battery/android` | Android `BatteryStatus`, `BatteryLevel` | sticky `ACTION_BATTERY_CHANGED` | status and level mapping stay here | no Java VM is `driver.ErrIncompatible`; no battery is `battery.ErrNoBattery` | import `x/ffi/native` |
| `x/driver/battery/darwin` | Darwin `BatteryStatus`, `BatteryLevel` | `AppleSmartBattery` via `ioreg` | status and level mapping stay here | not darwin is `driver.ErrIncompatible`; no battery is `battery.ErrNoBattery` | import `x/ffi` |
| `x/driver/media` | `Next`, `Previous`, `PlayPause`, `Stop`, `GetMetadata`, `Watch`, `StatusNotification` | MPRIS player | protocol stays here | no player is `ErrNoPlayer` | import `x/driver/audio_play`; post the alert here |
| `x/driver/power` | `Lock`, `Logout`, `Suspend`, `Hibernate`, `Reboot`, `Shutdown`, `Wake` | session power | protocol stays here | missing loginctl is `driver.ErrIncompatible` | import `x/ffi` |
| `x/driver/power/android` | Android `Lock`, `Suspend`, `Reboot`, `Shutdown` | binder power | the binder calls stay here | no binder is `driver.ErrIncompatible`; logout and hibernate are unavailable | import `x/ffi/native` |
| `x/driver/screen` | `SetDPMS`, `IsDPMSOn`, `ToggleDPMS`, `Reset` | display power | protocol stays here | missing swaymsg or xset is `driver.ErrIncompatible` | a hostname layout table |
| `x/driver/screen/android` | Android `SetDPMS`, `IsDPMSOn` | interactive bit | the binder calls stay here | no binder is `driver.ErrIncompatible`; reset is unavailable | import `x/ffi/native` |
| `x/driver/screenshot` | `Capture`, `SelectArea` | one image and a `wm.Rect` | protocol stays here | missing grim or maim is `driver.ErrIncompatible` | save into a config directory |
| `x/driver/wallpaper` | `SetStatic` | one still image | protocol stays here | missing feh or swaybg is `driver.ErrIncompatible` | import `x/ffi` |
| `x/driver/wm` | workspace switch, `AdvanceWorkspace`, `RotateWorkspaces`, scratchpad, focused rect, outputs | compositor IPC | protocol stays here | missing compositor is `driver.ErrIncompatible` | import `x/ffi`; post a notification here |
| `x/driver/camera` | `List`, `Capture` | one still frame | protocol stays here | missing ffmpeg or video device is `driver.ErrIncompatible` | import `x/ffi` |
| `x/driver/launcher` | `Choose`, `Prompt`, `Confirm`, `RunApp`, `SwitchWindow` | list, text, or yes/no | protocol stays here | missing menu tool is `driver.ErrUnavailable` | a file dialog; import `x/driver/filedialog` |
| `x/driver/terminal` | `Open`, `Options` | a terminal emulator | protocol stays here | missing emulator is `driver.ErrUnavailable` | import `x/ffi` |
| `x/driver/exec` | `Command`, `MustCommand`, `Run`, `RunProgram`, `Output`, `OutputString`, `Which`, `RequireBinary` | host process | the runner stays here | missing binary is `ErrNotFound`; `RequireBinary` is `driver.ErrIncompatible` | `os/exec.Command`; `os/exec.LookPath` |
| `x/driver/httpclient` | `Driver`, `Client`, `WithProgress` | process HTTP client | the client stays here | missing driver is the existing driver error | import `x/ffi` |
| `x/driver/fetchurl` | `Fetch`, `FetchOptions`, `StatusError` | hashed download | the protocol stays here | no URLs is `ErrNoURLs`; no writer is `ErrNoOutputWriter` | import `x/ffi` |
| `x/driver/treesitter` | `Get`, `Open`, `ForFile`, `Parse`, `Names`, `(*Tree).Parsed` | one grammar from a registered engine | protocol stays here | unknown language is `ErrUnknown`; no backend is `driver.ErrNotFound`; a null or error tree is `ErrParse` | import a grammar module |
| `x/driver/treesitter/ccgo` | ccgo registry | facade of ccgo-tree-sitter | selection stays here | a missing name is skipped | import a ccgo `grammar/<lang>` package |
| `x/driver/treesitter/leaven` | leaven registry | facade of leaven-tree-sitter | selection stays here | a missing name is skipped | import another tree-sitter module |
| `x/driver/treesitter/native` | installed `libtree-sitter-<name>` | facade of the tree-sitter binding | selection stays here | unset `LEWKIT_ENABLE_NATIVE_TREESITTER` or a missing library is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/treesitter/wazero` | wazero registry | facade of wazero-tree-sitter | selection stays here | a missing name is skipped | import a wazero `grammar/<lang>` package |
| `x/ndarray` | `Tensor`, ops, `Evaluator` | engine | ISA stays here | existing ndarray errors | import `x/ui/gui` |
| `x/ndarray/image` | pack `(h,w,4)` into `image.RGBA` | value | packing stays here | existing pack errors | hold bubbletea types; hold templ; hold transformers |
| `x/image` | CPU blit, `Label`, `RGB`, `BGR`, `CMYK`, `HSV` | value | blit stays here; each color space is its own struct and converts to `RGB` | existing blit errors | hold bubbletea types; hold templ; hold transformers |
| `x/image/convert` | `Decode`, `Pad`, `Resize`, `Square`, `EncodePNG`, `EncodeICO`, `EncodeICNS`, `ARGB` | value | icon bytes stay here | bad bytes are `ErrFormat` | import `x/driver` |
| `x/sound` | `Format`, `Mixer`, `Mix`, `Pipeline`, `Decode`, `Register`, `WriteWAV`, `ReadWAV` | PCM value | mixing, seek, and the decoder registry stay here | `ErrFormat`, `ErrFrame`, `ErrClosed`, `ErrSeek` | open a host device; import `x/driver`; import `x/sound/mp3`; import `x/sound/ogg` |
| `x/sound/mp3` | MP3 `Decoder` | registered decoder | decode stays here | mp3 decode error | import `x/driver` |
| `x/sound/ogg` | Ogg Vorbis `Decoder` | registered decoder | decode stays here | vorbis decode error | import `x/driver` |
| `x/sound/prelude` | blank import of the decoders | registry side effect | imports stay here | duplicate `Register` panics | decode by itself |
| `x/driver/audio_play` | `Open`, `Sinks`, `Config` | host sink is the playback writer | protocol stays here | missing driver is `driver.ErrUnavailable` | import `x/ffi/native`; import `x/ffi/wasm` |
| `x/ffi/native/pulse` | `Playback`, `List`, `Stream` | libpulse binding | simple playback stays here | pulse error text | import `x/ffi/wasm`; import `x/driver` |
| `x/driver/audio_play/pulse` | PulseAudio `Open` | facade of the pulse binding | selection stays here | missing library is `driver.ErrIncompatible` | import `x/ffi/native`; return `pa_simple` |
| `x/driver/audio_play/mem` | in-memory `Open` | test sink | capture stays here | incompatible unless `LEWKIT_ENABLE_MEMORY_DRIVER` is set | play on a host device |
| `x/driver/window/mem` | in-memory `Open` | test window | pixels stay here | incompatible unless `LEWKIT_ENABLE_MEMORY_DRIVER` is set | present on a host surface |
| `x/ffi/native/winmm` | `Open`, `Devices`, `Stream` | winmm binding | waveOut playback stays here | waveOut error text | import `x/ffi/wasm`; import `x/driver` |
| `x/driver/audio_play/winmm` | waveOut `Open` | facade of the winmm binding | selection stays here | missing library is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/ffi/native/coreaudio` | `Open`, `Devices`, `Stream` | AudioQueue binding | playback stays here | CoreAudio error text | import `x/ffi/wasm`; import `x/driver` |
| `x/driver/audio_play/coreaudio` | AudioQueue `Open` | facade of the coreaudio binding | selection stays here | missing framework is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/filedialog` | `Choose`, `Open`, `Request`, `Filter` | host file or folder dialog | protocol stays here | `ErrCanceled`, `ErrRequest`; a content URI with no document opener is `driver.ErrUnavailable`; missing driver is `driver.ErrUnavailable` | import `x/ffi/native`; import `x/ffi/wasm` |
| `x/driver/filedialog/portal` | `Choose`, `Available` | portal file chooser call | GTK and KDE service names stay here | missing bus name is `driver.ErrIncompatible` | import `x/ffi/native`; show a dialog by itself |
| `x/driver/filedialog/gtk` | GTK `Choose` | facade of the gtk portal backend | selection stays here | missing gtk portal is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/filedialog/qt` | Qt `Choose` | facade of the KDE portal backend | selection stays here | missing KDE portal is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/filedialog/cocoa` | `NSOpenPanel` and `NSSavePanel` | macOS file dialog | selection stays here | missing main thread is the file dialog error | import `x/ffi/wasm` |
| `x/driver/filedialog/win32` | common item dialog `Choose` | Windows file dialog | selection stays here | missing main thread is the file dialog error | import `x/ffi/wasm` |
| `x/driver/filedialog/android` | Android `Choose`, document `FS` | system document picker and document tree | selection stays here | no Java VM is `driver.ErrIncompatible`; cancel is `filedialog.ErrCanceled` | import `x/ffi/native` |
| `x/taskgroup/progress` | bubbletea viewer of `Session` | viewer of `Session` | stays next to `Session` | existing TUI skip rules | move into `x/ui/tui` |
| `x/ffi` | no Go API | names `native`, `wasm`, `android`, `jni` | MUST NOT grow a Go package | directory has no `.go` file | import `x/ffi` |
| `x/ffi/native` | `Open`, `OpenChain`, `OpenIn`, `SearchDirs`, `ProcOf`, `OpenFirst`, `Singleton`, `Once`, `Bind`, `Func`, `Symbol`, `Register`, `CString`, `GoString` | direct C ABI | loader stays here; one path is loaded once for a covered flag set; a soname chain is one singleton; Windows procedures use `ProcOf` | purego error; a failed `Open` is not cached; a failed `OpenChain` is cached | import `x/ffi/native/vulkan`; import `x/ffi/native/pulse`; import `x/ffi/native/winmm`; import `x/ffi/native/coreaudio`; import `x/ffi/native/treesitter`; import `x/ffi/native/android` |
| `x/ffi/wasm` | `Compile`, `Instance` | wasm runtime | host stays here | existing wasm errors | import `x/ffi/wasm/glsl`; import `x/ffi/wasm/capstone` |
| `x/ffi/native/vulkan` | `Device`, `Buffer`, `Shader`, `Cmd`, swapchain `Draw`, `OpenNative` | libvulkan binding | compute plus one graphics draw for a window the caller owns | existing vulkan errors | import `x/ffi/wasm`; import `x/driver`; open a host window |
| `x/ffi/wasm/glsl` | `Compile`, `CompileStage`, `CompileGlslang`, `Load`, `IsSPIRV`, `Hash`, `RegisterHash`, `Lookup` | glslang binding and SPIR-V registry | compiler and the hash registry stay here; a hit returns the stored SPIR-V; a miss compiles with the embedded glslang | `ErrCompile`, `ErrEmpty`, `ErrExist` | import `x/ffi/native` |
| `x/ffi/wasm/capstone` | `Open`, `Handle`, `Instruction` | Capstone binding | guest stays here | capstone error text | import `x/ffi/native` |
| `x/ffi/native/treesitter` | `Available`, `OpenLanguage`, `Parse` | libtree-sitter binding | loader stays here | missing library is the load error | import `x/ffi/wasm`; import `x/driver` |
| `x/ffi/native/webkitgtk` | `Load`, `Symbols` | WebKitGTK 6 and GTK 4, dlopen | loader stays here | missing library is `ErrUnavailable` | import `x/ffi/wasm`; listen on a port |
| `x/ffi/native/webkit` | `Load` | WebKit.framework, dlopen | loader stays here | missing framework is `ErrUnavailable` | import `x/ffi/wasm`; listen on a port |
| `x/ffi/native/webview2` | `Available`, `CreateEnvironment` | WebView2Loader.dll | loader stays here | missing loader is `ErrUnavailable` | import `x/ffi/wasm`; listen on a port |
| `x/ffi/native/android` | `JavaVMs`, `OnLooper` | libnativehelper and libandroid | loader stays here | missing library is the load error | import `x/driver` |
| `x/ffi/android` | `Open`, `Client` | `/dev/binder` session | one process-wide client | missing device is `ErrUnavailable` | import `x/driver`; import `x/ffi/native` |
| `x/ffi/jni` | `Bind`, `SetCurrentEnv`, `SetRunner`, `CallStatic`, `New`, `Class`, `StaticField`, `Field`, `Proxy`, `Ref`, `AsRef`, `Int`, `Text`, `Bool`, `Float`, `Context` | Java method, field, and interface proxy; `AsRef`, `Int`, `Text`, `Bool`, and `Float` decode a call result; `Context` is `lewkit.Host.app` | calls stay here; `Bind` keeps the app ClassLoader and JNIEnv from the Java thread that loaded the library; `SetRunner` runs later calls on that thread; `Proxy` uses one `lewkit.GoProxy` dispatcher; a float32 widens in `Float` | unbound loader, an ambiguous method, a missing member, or a Java exception is the error; a wrong Go type is `java value`; a null application context is an error; this build's stub is `ErrUnavailable` | import `x/driver`; import `x/ffi/native/android` |
| `x/driver/thread/jni` | Android looper factory | facade of the android binding | selection stays here | no Java looper is `driver.ErrIncompatible` | import `x/ffi/native` |
| `x/driver/webview` | `Open`, `View` | OS web view; page bytes and script messages stay in-process | protocol stays here | missing driver is the existing driver error | `net.Listen`; launch a browser; import `x/ffi/native` |
| `x/driver/vulkan` | `Open`, `List`, `OpenNative`, `Device` with `Buffer`, `Compile`, `Begin` | facade of the vulkan binding | selection stays here; `OpenNative` attaches a swapchain to a `window` the caller opened | existing vulkan errors | return the binding `Device`; import `x/ffi/native`; import `x/ffi/wasm`; `OpenScreen`; a second host window |
| `x/disasm` | `Engine`, object files, hex | facade of capstone | formats stay here | existing disasm errors | import `x/ffi/wasm` |
| `x/driver/ndeval` | CPU and Vulkan `Evaluator` factories | facade | factories stay here | existing ndarray errors | import `x/ffi/native/vulkan`; import `x/ffi/wasm` |
| `x/text/report` | `Finding`, `Format`, `Format.Render`, `WriteText`, `WriteTable`, `WriteRecords`, `WriteRustc`, `WriteSARIF` | diagnostic value | text, rustc, and SARIF stay here; the finding table is an `x/text/table` view | unknown format or level is the parse error; the zero `Format` is unset | import the root `report` package; import `x/ui`; import `x/driver` |
| `x/text/table` | `Format`, `Write`, `Column`, `Formatter`, `View`, `Make` | value | one writer for table, jsonl, and csv; `Make` builds a `View` once from a row type and a spec struct of `Field`s; a column spec picks order and replaces formats | unknown format is `ErrFormat`; a bad column is `ErrColumn` | import `x/cmd` |
| `examples` | programs, not a library | demo | demos MAY stay | program failure | import `examples` from a library package |

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
| INV-10 | `gui.Run` consumes a caller-supplied `Window`. `gui` MUST NOT call `window.Open`. `x/app` opens that window and calls `gui.Run`. | `x/ui/gui`; `x/app` | `gui.Open`; `EnsureDir` calling `window.Open` |
| INV-11 | This module is not a UI library | this repository | advertising `x/ui` as the product; a Flutter widget tree as the public API |
| INV-12 | Host window events include `Resize`, `Expose`, `Close`, `Pointer`, `Scroll`, `Key`, and `Drop` | `x/driver/window` | pointer `Msg` types that the host does not emit |
| INV-13 | `x/ffi` has no Go package | `x/ffi` | a `.go` file whose package is `ffi` |
| INV-14 | `x/ffi/native` does not import a nested binding | `x/ffi/native` | an import of `vulkan`, `pulse`, `winmm`, `coreaudio`, `webkitgtk`, `webkit`, `webview2`, or `android` |
| INV-15 | `x/ffi/wasm` does not import `x/ffi/wasm/glsl` | `x/ffi/wasm` | that import |
| INV-16 | `x/ffi/wasm` does not import `x/ffi/wasm/capstone` | `x/ffi/wasm` | that import |
| INV-17 | `x/ffi/native/vulkan` imports `x/ffi/native` and does not import `x/driver` | that package | an import of `x/ffi/wasm`; an import of `x/driver/window` or `x/driver/thread`; `OpenScreen` |
| INV-18 | `x/ffi/wasm/glsl` imports `x/ffi/wasm` | that package | an import of `x/ffi/native` |
| INV-19 | `x/ffi/wasm/capstone` imports `x/ffi/wasm` | that package | an import of `x/ffi/native` |
| INV-20 | `x/driver/vulkan.Device` does not return the binding device | `x/driver/vulkan` | a method whose result type is the binding `Device` |
| INV-21 | `x/driver/vulkan` does not import `x/ffi/native` | `x/driver/vulkan` | that import |
| INV-22 | `x/driver/ndeval` does not import `x/ffi/native/vulkan` | `x/driver/ndeval` | that import |
| INV-23 | `x/driver/ndeval` does not import `x/ffi/wasm` | `x/driver/ndeval` | that import |
| INV-24 | `x/disasm` does not import `x/ffi/wasm` | `x/disasm` | that import |
| INV-25 | `id_unix.go` loads libc through `x/ffi/native` | `x/driver/thread/id_unix.go` | an import of `x/ffi/wasm` |
| INV-26 | `x/driver/window/cocoa` loads frameworks through `x/ffi/native` | cocoa darwin files | an import of `x/ffi/wasm` |
| INV-27 | `main_darwin.go` loads `pthread_main_np` through `x/ffi/native` | `x/driver/thread/main_darwin.go` | an import of `x/ffi/wasm` |
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
| INV-46 | `x/driver/daynight` does not import `x/ui/gui`, `x/driver/webview`, or `x/ffi/wasm` | `x/driver/daynight` | that import |
| INV-47 | `x/driver/volume` does not import `x/driver/audio_play` | `x/driver/volume` | that import |
| INV-48 | `x/driver/launcher` does not import `x/driver/filedialog` | `x/driver/launcher` | that import |
| INV-49 | notification, clipboard, opener, dirs, share, volume, brightness, battery, media, power, screen, screenshot, wallpaper, wm, camera, launcher, and terminal do not import `x/ffi` | those packages | that import |
| INV-50 | `x/driver/treesitter` does not import `leaven-tree-sitter` | `x/driver/treesitter` | that import |
| INV-51 | `x/driver/treesitter/leaven` imports the leaven grammar module | `x/driver/treesitter/leaven` | a tree-sitter engine import in `x/driver/treesitter` |
| INV-52 | `x/driver/treesitter/ccgo` imports `ccgo-tree-sitter/core` and does not import a ccgo `grammar/<lang>` package | `x/driver/treesitter/ccgo` | that import |
| INV-53 | `x/driver/treesitter/native` imports `x/ffi/native/treesitter` and does not import `x/ffi/native` | `x/driver/treesitter/native` | an import of `x/ffi/native` |
| INV-54 | `x/ffi/native/treesitter` imports `x/ffi/native` and does not import `x/driver` | `x/ffi/native/treesitter` | an import of `x/driver` |
| INV-55 | `x/driver/treesitter/wazero` imports the wazero grammar module and does not import a wazero `grammar/<lang>` package | `x/driver/treesitter/wazero` | that import |
| INV-56 | `x/text/report` does not import the root `report` package, `x/ui`, or `x/driver` | `x/text/report` | that import |
| INV-57 | `x/text/table` does not import `x/cmd` | `x/text/table` | that import |
| INV-58 | `Open` reuses a handle when the cached flags cover the request. A failed `Open` is not cached. `OpenChain` tries each bare soname through `SearchDirs` once and keeps that result. `Singleton` and `Once` run on the caller goroutine | `x/ffi/native` | a second `dlopen` of the same path when the cached flags already cover the request; a second walk of the same soname chain |
| INV-59 | `x/release` does not import `x/driver` | `x/release` | that import |
| INV-60 | `x/driver/bundle` does not import `x/driver/webview` | `x/driver/bundle` | that import |
| INV-61 | `x/driver/thread/jni` imports `x/ffi/native/android` and does not import `x/ffi/native` | `x/driver/thread/jni` | an import of `x/ffi/native` |
| INV-62 | `x/ffi/native/android` imports `x/ffi/native` and does not import `x/driver` | `x/ffi/native/android` | an import of `x/driver` |
| INV-63 | `x/ffi/jni` imports `x/ffi/native` and does not import `x/driver` or `x/ffi/native/android` | `x/ffi/jni` | an import of `x/driver` or `x/ffi/native/android` |
| INV-64 | `x/driver/dirs/android`, `x/driver/power/android`, `x/driver/screen/android`, `x/driver/volume/android`, and `x/driver/daynight/android` import `x/ffi/android` and do not import `x/ffi/native` | those packages | an import of `x/ffi/native` |
| INV-65 | `x/driver/battery/android`, `x/driver/clipboard/android`, and `x/driver/brightness/android` import `x/ffi/jni` and `x/ffi/native/android` and do not import `x/ffi/native` | those packages | an import of `x/ffi/native` |
| INV-66 | `CompileStage` returns registered SPIR-V without starting the embedded glslang. A miss compiles with that reactor | `x/ffi/wasm/glsl` | starting the reactor before the registry lookup |
| INV-67 | `x/driver/filedialog/android` imports `x/ffi/jni` and `x/ffi/native/android` and does not import `x/ffi/native` | `x/driver/filedialog/android` | an import of `x/ffi/native` |

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
| `gui.EnsureDir` | empty dir and no terminal | Return `ErrNeedWindow`. The caller opens a window and calls `Pick`. |
| `gui.Model.View` | nil `Node` | Return `ErrView`. Do not `Draw`. |
| `filedialog.Choose` | `Save` with `Folder` or `Multiple` | Return `ErrRequest`. |
| `filedialog.Open` | no path | Return `ErrRequest`. |
| `filedialog.Open` | a content URI and a local path | Return `ErrRequest`. |
| `filedialog.Open` | a content URI and no document opener | Return `driver.ErrUnavailable`. |

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
3. Extract triangle and perlin from `examples/internal/scene` into `gui` only after they are reusable transformers.

## Assumptions

| ID | Fact | If false |
|----|------|----------|
| AS-01 | bubbletea v2 remains the tui toolkit | Change the tui tooling row. Do not invent a terminal runtime. |
| AS-02 | templ remains the intended web toolkit | Change the web later-work item before adding `x/ui/web`. |
| AS-03 | Pointer coordinates are client pixels with the same origin as `Frame` | Change the cocoa Y flip / scale if a host uses another origin. |

## Decision history

- 2026-09-20 grill: native export placement; SPEC at repo root. Rejected: nested `x/ui/SPEC.md`; a shared `Widget`; moving progress into `tui`; moving triangle into `gui` now; calling this a UI library.
- 2026-10-01: demos left `cmd/lewkit/experiments` for one program per directory under `examples/`. Each program has `eletrocromo.json` and calls `x/entry`. A library package MUST NOT import `examples`. Rejected: moving triangle, perlin, and compute into `x/ui/gui`.
- 2026-10-01: `BatteryLevel` is an integer from 0 to 100 on the same supply as `BatteryStatus`. Linux reads `capacity`, then `energy_now`/`energy_full`, then `charge_now`/`charge_full`. Android reads `EXTRA_LEVEL` and `EXTRA_SCALE` on the sticky battery intent. No supply is `ErrNoBattery`. A supply with no level is `ErrUnknownLevel`.
- 2026-10-01: Darwin reads `AppleSmartBattery` through `ioreg`. `CurrentCapacity` and `MaxCapacity` become the 0..100 level. No registry entry is `ErrNoBattery`. Linux sysfs stays the Linux backend.
- 2026-09-20: `gui` follows bubbletea (`Model` / `Msg` / `Cmd` / `Run`) with tensors in `View`. Host events are `Resize`, `Expose`, `Close` only. Rejected: Flutter widget tree as the public API; `gui` calling `window.Open`.
- 2026-09-21: `gui.Model.View` returns a layout `Node`. `Run` paints it through `Picture` to a `(h,w,4)` tensor.
- 2026-09-20: window bus adds `Pointer`, `Scroll`, and `Key`. `gui.Run` forwards them. Marquee drag/wheel/space.
- 2026-09-24: window bus adds `Drop` (local paths). X11 delivers it through Xdnd. Cocoa delivers it through `NSDraggingDestination` on the content view. `gui.Run` forwards it with the other host events.
- 2026-09-21: C libraries live under the mechanism that loads them. `x/ffi/native/vulkan`, `x/ffi/wasm/glsl`, `x/ffi/wasm/capstone`. `x/driver/vulkan`, `x/driver/ndeval`, and `x/disasm` are facades. `x/ffi` is not a Go package. `x/thread` and cocoa call `x/ffi/native`.
- 2026-09-23: templ is adopted. A tag for one registered asset lives in that asset package. Page templates stay in `x/ui/web`. htmx, tailwindcss, jquery, and sakuracss are blank-import assets served from `/__lewkit__/`. The first page template is `x/ui/web` `Page`.
- 2026-10-02: daisyUI 5.6.18 is a blank-import asset next to the vendored Tailwind browser build. Its tag is `x/http/asset/daisyui` `Load`. Pages load that tag instead of a CDN URL.
- 2026-10-02: `filedialog.Open` reads paths from `Choose` as an `fs.FS`. One directory is the root. Several paths share a root named by base name. A `content:` URI is the Android document tree. The bytes stay in the provider.
- 2026-10-03: `SurfaceActivity` records `Host.foreground` on resume and clears it on pause, same as the splash and page activities. The document picker starts from that activity.
- 2026-10-03: `examples/welcome` serves a page before `filedialog.Choose`. A GUI window as the first activity never writes the Android ready line, so the splash stays on "Starting local server…".
- 2026-09-24: light or dark is `x/driver/daynight`. Web views push it into the page without a reload. `gui.Run` delivers `ModeMsg`.
- 2026-09-25: host capabilities that lived in modot and eletrocromo sit under `x/driver`. Volume is sink level, not PCM playback. Launcher is a menu, not a file dialog. Termux backends stay in modot. Screen reset enables outputs; it does not store a hostname layout. Share does not own the eletrocromo JSONL drop.
- 2026-09-25: status alerts, workspace rotation, the next-workspace counter, and Wake-on-LAN live in the driver packages. A status function returns the alert. The caller posts it. A change function does not post. Screenshot still returns an image. The caller saves it.
- 2026-09-26: `x/text/report` owns diagnostic findings. Output is a text line, a table, a rustc-style snippet, or SARIF 2.1.0. The root `report` package stays the error-reporter registry. The finding table is an `x/text/table` view, so the same columns render as a table, JSONL, or CSV.
- 2026-09-26: the reverse-domain id is `x/release.AppID`. `x/driver/bundle` is that binary's data, cache, config, and web profile. `x/driver/dirs` stays the generic tree. `lewkit build` writes desktop archives and the Android, macOS, and iOS hosts. A web view request is normalized inside `webview.Dispatch`.
- 2026-10-02: `x/ui/gui` has no `Open`. `x/app` calls `window.Open` and then `gui.Run`. `EnsureDir` returns `ErrNeedWindow` when a picker is required. `Pick` runs `Welcome` on the caller's window.
- 2026-10-02: `x/driver/window` is the only host window. `vulkan.OpenNative` attaches a swapchain to that window. `OpenScreen` and `x/driver/vulkanwindow` are removed. The binding takes a UI thread hook from `x/driver/vulkan` and does not import `x/driver`.
- 2026-10-02: `x/release.version` is the only runtime version stamp. `x/build/version` keeps packaging `Info` and does not export `Version`. Git describe, rev-parse, log, and rev-list go through `x/git`.
- 2026-10-02: Pad, resize, PNG, ICO, and ICNS encoding live in `x/image/convert`. `x/build/icons` keeps knockout, the upper-mark crop, and the packaging file tree.
- 2026-10-02: `Triangle` and `TriangleTurn` leave `x/image`. The RGB triangle demo stays in `examples/internal/scene`.
- 2026-10-02: name-and-args command runs live on `x/driver/exec` as `RunProgram` and `OutputString`. `x/driver/exec`, `x/driver/httpclient`, and `x/driver/fetchurl` have placement rows. Android volume, power, and screen have rows.
- 2026-10-02: `x/io.Mkdirp` is removed. The profile writer uses `os.MkdirAll`.
- 2026-10-02: `x/future` is removed. It had no production caller.
- 2026-10-02: `sqlite3`, `file`, and `postgresql` are not separate connectors. `splitURL` maps them onto `sqlite` and `postgres`.
- 2026-10-02: `lewkit` launch extracts the first archive member through `x/fs/tar.ExtractFirst` and `x/fs/zip.ExtractFirst`.
- 2026-10-03: `x/fs.Index` decorates a flat listing with directory lookup. A later file replaces an earlier one. A later directory keeps the newer ModTime. A file and a directory at the same name is `fs.ErrExist`. Tar and compose use it. UDF, WIM, zip, and squashfs walk the format's own directories and do not keep a second index. `x/fs.Lookup` is that walk.
- 2026-10-03: `x/entry` starts the task session after the command is parsed. `MainFrom` keeps context values, including pool caps from `taskgroup.WithLimits`. `Main` still uses `DefaultLimits`.
- 2026-10-03: the short product name is `x/release.Name`. The stamp `-X github.com/lewtec/lewkit/x/release.name` wins, then `LEWKIT_NAME`, then the built-in default. An empty or invalid override keeps that default. Paths, protocol names, and titles call `Name`. A caller's own field still overrides it. `AppID` stays the reverse-domain id. The Java package `lewkit` and the asset URL `/__lewkit__/` stay fixed contracts.
- 2026-10-03: `x/driver/android` is removed. It was not a capability. Java result decoding and the `lewkit.Host` application context live in `x/ffi/jni`. The process binder client stays `x/ffi/android.ForAndroid`. Android dirs, power, screen, volume, and daynight map a binder failure to `driver.ErrIncompatible`. A missing application context stays `driver.ErrUnavailable` at the driver that reads it.
