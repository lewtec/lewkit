package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgsBasic(t *testing.T) {
	type BasicArgs struct {
		name    StringArg
		idade   IntArg[uint]
		verbose Count `short:"v"`
	}

	var cmd Command[BasicArgs]

	assert.NoError(t, cmd.Parse("--name", "Lucas", "--idade", "26", "-vvv"))
	assert.Equal(t, cmd.args.name.Value(), "Lucas")
	assert.Equal(t, cmd.args.idade.Value(), 26)
	assert.Equal(t, cmd.args.verbose.Value(), 3)
}
