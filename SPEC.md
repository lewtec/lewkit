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
- `path:cmd/lewkit/experiments`: triangle, perlin, compute demos.
- templ is absent from `go.mod`.

## Technique

| ID | Input | Rule | Output |
|----|-------|------|--------|
| TEC-01 | A new type | Classify by native export. The caller-imported type picks the owner in the placement table. A type that uses a toolkit to view another package's type stays next to that type. | Exactly one owner package |
| TEC-02 | A type that matches two of `tui`, `web`, `gui` | Split it into two types. Each type has one native export. | No shared `Widget` in `x/ui` |
| TEC-03 | A type under `x/ui` | The type is a component. The application opens the host. `gui.Run` drives a supplied `Window` the way `tea.Program` drives a terminal. | `window.Open` stays outside `gui` |
| TEC-04 | An import edge | `tui`, `web`, and `gui` MAY import host and engine. Host and engine MUST NOT import them. | Acyclic ownership |

## Tooling

| TEC | Tool | Relation | We do not | Cite |
|-----|------|----------|-----------|------|
| TEC-01 | this SPEC | implement | add a fourth package under `x/ui` | none |
| TEC-02 | this SPEC | implement | unify `tui`, `web`, and `gui` behind one interface | none |
| TEC-03 | existing host loops | wrap | start the host loop from `x/ui` packages | path:x/driver/window path:x/taskgroup/progress |
| TEC-04 | ndarray + window | wrap | relocate Present into `gui`; relocate Tensor into `gui` | path:x/ndarray path:x/driver/window |
| TEC-01 tui | bubbletea v2 | adopt | write a terminal runtime | go.mod path:x/taskgroup/progress |
| TEC-01 gui | ndarray 21-op graph | wrap | add a rasterizer beside ndarray | path:x/ndarray |

| Cell | Pick | C or D | Implements | Cite if C |
|------|------|--------|------------|-----------|
| Language | Go 1.27 | C | all | go.mod |
| Runtime | the calling Go process | C | TEC-03 | go.mod |
| Persistence | none | C | this SPEC stores no data | none |
| UI | not a UI library; three component packages under `x/ui` | D | TEC-01 | |
| Packaging | this Go module | C | all | go.mod |
| Identity | none | C | packages have no user identity | none |
| Host OS | window backends already in tree | C | TEC-03 TEC-04 | path:x/driver/window |

templ is later work. It is not an adopted tool in this module.

## Terminology

| Concept | Approved | Banned |
|---------|----------|--------|
| primitive | a reusable type in this module | UI library, widget toolkit, Flutter |
| namespace | `x/ui` as a directory of three packages | parent `Widget`; importable `x/ui` API |
| component | bubbletea type; templ template; tensor transformer | widget |
| transformer | `Picture.Render` of a `Node` tree: a fused `(h,w,4)` `Tensor` | shader, painter, widget, GUI framework |
| Model | Init, Update, View (bubbletea shape; View is a layout Node) | Widget, Flutter Element |
| host | `x/driver/window` | gui, windowing toolkit |
| engine | `x/ndarray` | tinygrad |
| viewer | code that uses a toolkit to show another package's type | component package |
| demo | `cmd/lewkit/experiments` | component package |

`tui`, `web`, and `gui` are package names. They are not a product.

## Types

| Type | Exported | Identity or value | Mutable | Nil/error | Callers MUST NOT |
|------|----------|-------------------|---------|-----------|------------------|
| `x/ui` | no Go API | names `tui`, `web`, `gui` | MUST NOT grow types | the directory MAY have no Go package | import `x/ui` |
| `x/ui/tui` | bubbletea types for callers to compose | value | catalog MAY grow | package MAY be absent until the first type | put Session viewers here |
| `x/ui/web` | templ templates | value | catalog MAY grow | package MAY be absent until the first template | put templ outside `web` |
| `x/ui/gui` | `Model`, `Msg`, `Cmd`, `Run`; `View` is a layout `Node` | value | catalog MAY grow | package MAY be absent until the first transformer | own the host; own the engine; call `window.Open` |
| `x/driver/window` | `Open`, `Frame`, `Fit`, `Present`, `Animate` | host identity is the opened window | protocol stays here | missing driver is the existing window error | move Present into `gui` |
| `x/ndarray` | `Tensor`, ops, `Evaluator` | engine | ISA stays here | existing ndarray errors | import `x/ui/gui` |
| `x/ndarray/image` | pack `(h,w,4)` into `image.RGBA` | value | packing stays here | existing pack errors | hold bubbletea types; hold templ; hold transformers |
| `x/image` | CPU blit, `Label` | value | blit stays here | existing blit errors | hold bubbletea types; hold templ; hold transformers |
| `x/taskgroup/progress` | bubbletea viewer of `Session` | viewer of `Session` | stays next to `Session` | existing TUI skip rules | move into `x/ui/tui` |
| `cmd/lewkit/experiments` | commands, not a library | demo | demos MAY stay | command failure | import experiments as a component package |

## Invariants

| ID | Predicate | On | Forbidden bypass |
|----|-----------|----|------------------|
| INV-01 | A type has exactly one owner in the placement table | every new type | a fourth package under `x/ui`; a type copied into two owners |
| INV-02 | `x/ui` exports no types | `x/ui` | `Widget`, shared `Color`, shared `Align` |
| INV-03 | `tui` native export is a bubbletea type | `x/ui/tui` | templ files; tensor transformers |
| INV-04 | `web` native export is a templ template | `x/ui/web` | bubbletea types; tensor transformers |
| INV-05 | `gui.Model.View` returns a layout `Node`; `Run` paints it to a `(h,w,4)` tensor | `x/ui/gui` | `View` returning a tensor; `window.Present` declared here; a second rasterizer |
| INV-06 | A viewer stays next to the type it shows | `x/taskgroup/progress` | move progress into `x/ui/tui` because it uses bubbletea |
| INV-07 | `tui`, `web`, and `gui` MUST NOT import each other | those packages | `gui` emitting HTML; `tui` importing `gui` |
| INV-08 | Host and engine MUST NOT import `tui`, `web`, and `gui` | `x/driver/window`, `x/ndarray` | `window` depending on `gui` |
| INV-09 | This repository has one constitution: `SPEC.md` at the repo root | this file | `x/ui/SPEC.md`; a second SPEC beside this file |
| INV-10 | `gui.Run` consumes a caller-supplied `Window`. It MUST NOT call `window.Open`. | `x/ui/gui` | `gui` opening a host window |
| INV-11 | This module is not a UI library | this repository | advertising `x/ui` as the product; a Flutter widget tree as the public API |
| INV-12 | Host window events include `Resize`, `Expose`, `Close`, `Pointer`, `Scroll`, and `Key` | `x/driver/window` | pointer `Msg` types that the host does not emit |

## Errors

| Public operation | Bad input | One reaction |
|------------------|-----------|--------------|
| Place a type | Matches two of `tui`, `web`, `gui` | Split into two types. MUST NOT add a `Widget` in `x/ui`. |
| Place a type | Matches no row in the table | Leave it in its existing owner. MUST NOT add a fourth package under `x/ui`. |
| Place a type | Uses bubbletea to view `Session` | Keep it in `x/taskgroup/progress`. |
| Place a type | First templ template in the module | Create `x/ui/web`. MUST NOT put the file in `gui`. MUST NOT put the file in `tui`. |
| Place a type | First reusable tensor transformer for a pixel frame | Create `x/ui/gui`. MUST NOT leave it in `x/ndarray`. MUST NOT leave it in `x/driver/window`. |
| Export from `x/ui` | A Go type on the namespace | Move the type into the one of `tui`, `web`, `gui` that needs it. |
| `gui.Run` | nil `Model` | Return `ErrModel`. |
| `gui.Run` | nil `Window` | Return `window.ErrClosed`. |
| `gui.Model.View` | nil `Node` | Return `ErrView`. Do not `Draw`. |

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

## Later work

1. Reusable bubbletea types in `x/ui/tui`.
2. Adopt templ in this module and add templates in `x/ui/web`.
3. Text input (IME) and mapped key names.
4. Extract triangle and perlin from experiments into `gui` only after they are reusable transformers.

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
