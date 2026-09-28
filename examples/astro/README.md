# Astro example

This example serves an Astro server-rendered page in a lewkit web window.
Each request loads one cat fact from catfact.ninja.
The page has no client JavaScript and no offline fallback.

The Go binary embeds the worker script and the client assets.
Build that embed before you run the app.
The process does not run esbuild at startup.

## Build the embed

1. `mise run astro` builds the Astro site into `dist/`.
2. `mise run assemble` copies that build into `worker/`.
3. `mise run embed` writes `embed/guest.js` and `embed/assets/`.

`mise run build` runs all three steps.

## Run the window

From the repository root:

```bash
go run ./cmd/lewkit release run --config ./examples/astro/eletrocromo.json
```

`mise run run` in this directory runs `go run .`.
That command panics. A plain `go run` has no release stamp.

## Requirements

Install tools with mise. `mise.toml` pins bun.
The module dependency is `github.com/lucasew/orvalho`.
