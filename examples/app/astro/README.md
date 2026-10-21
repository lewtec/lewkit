# Astro example

This directory is an Astro site. `entry` is `astro:.`.

`lewkit release run` installs the packages, builds the site, bundles the worker, and opens the window. `bun.lock` selects bun. The release installs bun when that program is missing. The page requests one fact from catfact.ninja on each visit.

## Run

From the repository root:

```bash
go run ./cmd/lewkit release run --config ./examples/app/astro/eletrocromo.json
```
