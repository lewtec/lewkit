package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
)

func TestBuildAllParses(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "build-all", "--out", "dist", "--archive", "demo", "--id", "br.tec.lew.demo", "--name", "Demo")
	assert.Equal(t, "dist", app.Args.release.buildAll.out.Value())
	assert.Equal(t, "demo", app.Args.release.buildAll.file.Value())
	assert.Equal(t, "br.tec.lew.demo", app.Args.release.buildAll.id.Value())
	assert.Equal(t, "Demo", app.Args.release.buildAll.name.Value())
}
