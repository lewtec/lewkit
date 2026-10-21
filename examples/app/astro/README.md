# Astro example

This directory is an Astro site. `entry` is `astro:.`.

`lewkit release run` builds the site, bundles the worker, and opens the window. The page requests one fact from catfact.ninja on each visit. The build does not install packages.

## Install

Run this command in this directory:

```bash
mise run install
```

`bun install` reads `bun.lock`.

## Run

Run this command in this directory:

```bash
mise run run
```

From the repository root, `bun` must be on `PATH`:

```bash
go run ./cmd/lewkit release run --config ./examples/app/astro/eletrocromo.json
```
