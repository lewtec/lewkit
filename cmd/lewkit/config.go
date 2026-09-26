package main

import (
	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/cmd"
)

// appConfig is the flag set shared by release build and release run.
// Flags overlay eletrocromo.json.
type appConfig struct {
	goos    goosArg       `long:"goos" help:"target GOOS"`
	goarch  goarchArg     `long:"goarch" help:"target GOARCH"`
	dir     cmd.StringArg `help:"module directory" default:"."`
	id      cmd.StringArg `long:"id" help:"reverse-domain app id"`
	name    cmd.StringArg `long:"name" help:"app name"`
	version cmd.StringArg `long:"version" help:"version name" default:""`
	config  cmd.StringArg `long:"config" help:"eletrocromo.json file or directory" default:""`
	main    cmd.StringArg `long:"main" help:"main package directory" default:""`
}

func (c appConfig) spec() build.Spec {
	return build.Spec{
		Dir:     c.dir.Value(),
		Config:  c.config.Value(),
		ID:      c.id.Value(),
		Name:    c.name.Value(),
		Version: c.version.Value(),
		GoMain:  c.main.Value(),
	}
}
