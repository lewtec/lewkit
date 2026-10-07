package main

import (
	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/sops"
)

// appConfig is the flag set shared by release build and release run.
// Flags overlay eletrocromo.json.
type appConfig struct {
	goos        goosArg       `long:"goos" help:"target GOOS"`
	goarch      goarchArg     `long:"goarch" help:"target GOARCH"`
	dir         cmd.StringArg `help:"module directory" default:"."`
	id          cmd.StringArg `long:"id" help:"reverse-domain app id" default:""`
	name        cmd.StringArg `long:"name" help:"app name" default:""`
	version     cmd.StringArg `long:"version" help:"version name" default:""`
	config      cmd.StringArg `long:"config" help:"eletrocromo.json file or directory" default:"./eletrocromo.json"`
	main        cmd.StringArg `long:"main" help:"main package directory" default:""`
	p12         sops.File     `long:"p12" env:"LEWKIT_SIGN_P12" help:"PKCS#12 publisher key file. A SOPS age file is decrypted." default:""`
	p12Password sops.File     `long:"p12-password" env:"LEWKIT_SIGN_P12_PASSWORD" help:"file containing the PKCS#12 password. A SOPS age file is decrypted." default:""`
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
