package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type BasicArgs struct {
	name    StringArg    `long:"name" short:"n"`
	idade   IntArg[uint] `long:"idade" short:"i"`
	verbose Count        `short:"v" long:"verbose"` // -vvv or --verbose 3
	rest    []StringArg  // that means a positional argument
}

func (a *BasicArgs) Run(ctx context.Context) error {
	fmt.Printf("name is %s, age is %d", a.name.Value(), a.idade.Value())
	for _, arg := range a.rest {
		fmt.Printf("- %s", arg.Value())
	}
	return nil
}
func TestArgsBasic(t *testing.T) {

	var cmd Command[BasicArgs]

	assert.NoError(t, cmd.Parse("--name", "Lucas", "--idade", "26", "-vvv"))
	assert.Equal(t, cmd.args.name.Value(), "Lucas")
	assert.Equal(t, cmd.args.idade.Value(), 26)
	assert.Equal(t, cmd.args.verbose.Value(), 3)
}
