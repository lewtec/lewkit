package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/cmd/lewkit/decomp"
	"github.com/lewtec/lewkit/cmd/lewkit/generate"
	"github.com/lewtec/lewkit/cmd/lewkit/term"
	"github.com/lewtec/lewkit/report/sentry"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"

	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	_ "github.com/lewtec/lewkit/x/tool/conda"
)

func main() {
	if done, err := toolchainChild(); done {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	app, err := cmd.Parse[cmd.App[root]](normalizeReleaseGoFlags(cmd.Args())...)
	if err != nil {
		entry.Main(context.Background(), func(context.Context) error { return err })
		return
	}
	parent := taskgroup.WithLimits(context.Background(), app.Args.Apply(taskgroup.DefaultLimits()))
	entry.MainFrom(parent, app.Run)
}

type root struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
	sentry        sentry.Arg `long:"sentry-dsn" env:"SENTRY_DSN" help:"Sentry DSN" default:"https://26fa6b84edbc334b77bf7f6e1d7d69bc@o4508616651505664.ingest.us.sentry.io/4512090764607488"`
	generate      *generate.Cmd
	release       *releaseCmd
	exec          *execCmd
	disasm        *disasmCmd
	decomp        *decomp.Cmd
	doctor        *doctorCmd
	workflow      *workflowCmd
	completion    *completionCmd
	term          *term.Cmd
}

func (r *root) Setup() error {
	return r.sentry.Setup()
}

type completionCmd struct{}

func (completionCmd) Description() string {
	return "print the bash complete -C line"
}

func (*completionCmd) Run(context.Context) error {
	_, err := fmt.Println(cmd.BashCompleteLine(cmd.Name()))
	return err
}

func (root) Description() string {
	return "Well planned primitives to be used in other projects."
}
