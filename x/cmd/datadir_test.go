package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataDirArg(t *testing.T) {
	type args struct {
		dir DataDirArg `long:"dir"`
	}
	runDirArgCases(t, func(in string) (args, error) {
		return Parse[args]("--dir", in)
	}, func(got args) string {
		return got.dir.Value()
	})
}

func TestDataDirDefault(t *testing.T) {
	type args struct {
		dir DataDirArg `long:"dir" default:"."`
	}
	got := ParseOK[args](t)
	assert.Equal(t, ".", got.dir.Value())
}
