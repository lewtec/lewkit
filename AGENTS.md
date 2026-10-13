- Optimize code for human readers. High signal/noise ratio.
- Avoid shortening names
- Keep stuff simple and organized, do step backs and refactors if necessary
- Respect the context chain: only context.Background in main, ctx as first positional arg
- Table all errors: %w in fmt.Errorf
- Explicit is better than implicit
- Follow the patterns already in the repo: x/cmd, x/test, testify, x/db with sqlc and migrations

## Cloud Agent

Tools are pinned in `mise.toml`. The environment install puts Go 1.27, `modot`, `goreleaser`, and `svu` on `PATH`, and sets `CGO_ENABLED=0` through mise.

- Build the program with `go build -o lewkit ./cmd/lewkit`.
- Test with `mise run test`. That runs `x/toolchain/test.sh` and skips `x/toolchain/go/src`. `go test ./...` loads the vendored compiler and is the command that exhausted CI.
- `github.com/lewtec/lewkit/cmd/lewkit` tests need a 40 minute timeout. `TestReleaseGoLewkit` builds this module with the embedded compiler and skips under `-short`.
- Lint with `mise run lint`. Format with `mise run format`.
