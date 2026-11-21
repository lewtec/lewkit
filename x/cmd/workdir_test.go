package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkDirArg(t *testing.T) {
	type args struct {
		dir WorkDirArg `short:"C" long:"directory"`
	}
	runDirArgCases(t, func(in string) (args, error) {
		return Parse[args]("-C", in)
	}, func(got args) string {
		return got.dir.Value()
	})
}

func TestWorkDirDefault(t *testing.T) {
	type args struct {
		dir WorkDirArg `short:"C" long:"directory"`
	}
	got := ParseOK[args](t)
	assert.Equal(t, ".", got.dir.Value())
}
