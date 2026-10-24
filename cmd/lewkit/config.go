package main

import (
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/host"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/sops"
)

// appConfig is the flag set shared by release build and release run.
// A set flag replaces that field of eletrocromo.json.
// An omitted or blank flag leaves the file value.
// An omitted --config reads ./eletrocromo.json, or <dir>/eletrocromo.json when --dir is set.
// That file can be missing when a field flag is set. An explicit --config is required.
type appConfig struct {
	goos    goosArg       `long:"goos" help:"target GOOS"`
	goarch  goarchArg     `long:"goarch" help:"target GOARCH"`
	dir     cmd.StringArg `long:"dir" help:"app directory when --config is omitted" default:"."`
	id      cmd.StringArg `long:"id" help:"reverse-domain app id" default:""`
	name    cmd.StringArg `long:"name" help:"app name" default:""`
	version cmd.StringArg `long:"version" help:"version name" default:""`
	config  cmd.StringArg `long:"config" help:"eletrocromo.json file or directory" default:"./eletrocromo.json"`
	entry   cmd.StringArg `long:"entry" help:"go:<dir>, astro:<dir>, or static:<zip-or-dir>" default:""`
	main    cmd.StringArg `long:"main" help:"main package directory" default:""`
	icon    cmd.StringArg `long:"icon" help:"master PNG or JPEG" default:""`
	p12     sops.File     `long:"p12" env:"LEWKIT_SIGN_P12" help:"passwordless PKCS#12 publisher key. A SOPS age file is decrypted." default:""`
}

func (c appConfig) spec() host.Spec {
	spec := host.Spec{
		Dir:     strings.TrimSpace(c.dir.Value()),
		ID:      strings.TrimSpace(c.id.Value()),
		Name:    strings.TrimSpace(c.name.Value()),
		Version: strings.TrimSpace(c.version.Value()),
		Entry:   strings.TrimSpace(c.entry.Value()),
		GoMain:  strings.TrimSpace(c.main.Value()),
		Icon:    strings.TrimSpace(c.icon.Value()),
	}
	if c.config.ArgSet() {
		spec.Config = strings.TrimSpace(c.config.Value())
		return spec
	}
	if overlay(spec) {
		return spec
	}
	if c.dir.ArgSet() {
		dir := spec.Dir
		if dir == "" {
			dir = "."
		}
		spec.Config = filepath.Join(dir, apk.ConfigFileName)
		return spec
	}
	spec.Config = c.config.Value()
	return spec
}

func overlay(spec host.Spec) bool {
	return spec.ID != "" || spec.Name != "" || spec.Version != "" || spec.Entry != "" || spec.GoMain != "" || spec.Icon != ""
}
