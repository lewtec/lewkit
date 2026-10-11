package ndeval

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

type evalLeaf struct {
	ev ndarray.Evaluator
}

func (l *evalLeaf) Run(ctx context.Context) error {
	ev, ok := cmd.Lookup[ndarray.Evaluator](ctx, "gpu")
	if !ok || ev == nil {
		return errors.New("missing evaluator")
	}
	l.ev = ev
	return nil
}

type evalRoot struct {
	GPU  Arg `long:"gpu" default:"cpu" ctx:"gpu"`
	Leaf *evalLeaf
}

func TestParseExpr(t *testing.T) {
	got, err := parseExpr(" vulkan:integrated | cpu ")
	require.NoError(t, err)
	require.Equal(t, []choice{{name: "vulkan", kind: "integrated"}, {name: "cpu"}}, got)
	_, err = parseExpr("opengl:integrated")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
	_, err = parseExpr("|")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
	_, err = parseExpr("vulkan:nope")
	require.ErrorIs(t, err, cmd.ErrInvalidArgument)
}

func TestEvalCPU(t *testing.T) {
	ev, err := Eval(t.Context(), "cpu")
	require.NoError(t, err)
	require.Equal(t, ndarray.CPU, ev)
}

func TestArgGivesOneEvaluator(t *testing.T) {
	app, err := cmd.Parse[cmd.App[evalRoot]]("leaf", "--gpu", "cpu")
	require.NoError(t, err)
	require.NoError(t, app.Run(t.Context()))
	require.Equal(t, ndarray.CPU, app.Args.Leaf.ev)
}
