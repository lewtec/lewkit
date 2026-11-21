package taskgroup

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
)

// Arg is the task session. Flatten-embed it so --io/--cpu/--internet
// land on the parent. ctx stores the Session after Enter (or the Arg
// itself before). Zero flags keep DefaultLimits (or the base passed
// to Enter).
type Arg struct {
	IO       cmd.IntArg[int] `long:"io" help:"max concurrent IO tasks (0 keeps config/default)" default:"0"`
	CPU      cmd.IntArg[int] `long:"cpu" help:"max concurrent CPU tasks (0 keeps config/default)" default:"0"`
	Internet cmd.IntArg[int] `long:"internet" help:"max concurrent network tasks (0 keeps config/default)" default:"0"`

	session *Session
}

func (a Arg) Apply(base Limits) Limits {
	if n := a.IO.Value(); n > 0 {
		base.IO = n
	}
	if n := a.CPU.Value(); n > 0 {
		base.CPU = n
	}
	if n := a.Internet.Value(); n > 0 {
		base.Internet = n
	}
	return base
}

// Value is the session started by Enter, or nil before that.
func (a Arg) Value() *Session {
	return a.session
}

// Enter starts a Session with flags applied on top of base.
// It does not start the progress TUI; that happens later, lazily,
// when progress.Run sees the first Go or LineWriter.
func (a *Arg) Enter(ctx context.Context, base Limits) (*Session, context.Context) {
	if a.session != nil {
		return a.session, context.WithValue(ctx, sessionKey{}, a.session)
	}
	s, ctx := New(ctx, a.Apply(base))
	a.session = s
	return s, ctx
}
