package common

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestMaterializeHost_RendersAndWritesJSON(t *testing.T) {
	src := fstest.MapFS{
		"template/Info.plist.tmpl": {Data: []byte(`<name>{{.Name}}</name>`)},
	}
	out := filepath.Join(t.TempDir(), "host")
	err := MaterializeHost(src, struct{ Name string }{Name: "App"}, out, false, []byte("{\"ok\":true}\n"))
	if err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plist) != "<name>App</name>" {
		t.Fatalf("plist: %q", plist)
	}
	jsonb, err := os.ReadFile(filepath.Join(out, HostConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(jsonb) != "{\"ok\":true}\n" {
		t.Fatalf("json: %q", jsonb)
	}
}

func TestMaterializeHost_EmptyOutDir(t *testing.T) {
	src := fstest.MapFS{
		"template/static.txt": {Data: []byte("x")},
	}
	if err := MaterializeHost(src, nil, "  ", false, nil); !errors.Is(err, ErrOutDirRequired) {
		t.Fatalf("want ErrOutDirRequired, got %v", err)
	}
}

func TestWalkTemplate_RendersAndCopies(t *testing.T) {
	src := fstest.MapFS{
		"template/Info.plist.tmpl": {Data: []byte(`<name>{{.Name}}</name>`)},
		"template/static.txt":      {Data: []byte("keep")},
		"template/settings.kts":    {Data: []byte(`name = "{{.Name}}"`)},
		"template/scripts/run.sh":  {Data: []byte("#!/bin/sh\n")},
	}
	out := t.TempDir()
	if err := WalkTemplate(src, struct{ Name string }{Name: "App"}, out); err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plist) != "<name>App</name>" {
		t.Fatalf("plist: %q", plist)
	}
	static, err := os.ReadFile(filepath.Join(out, "static.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(static) != "keep" {
		t.Fatalf("static: %q", static)
	}
	kts, err := os.ReadFile(filepath.Join(out, "settings.kts"))
	if err != nil {
		t.Fatal(err)
	}
	if string(kts) != `name = "App"` {
		t.Fatalf("kts: %q", kts)
	}
	info, err := os.Stat(filepath.Join(out, "scripts/run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("run.sh not executable: %v", info.Mode())
	}
}

func TestWalkTemplateDest_RemapsPath(t *testing.T) {
	src := fstest.MapFS{
		"template/app/src/main/kotlin/Main.kt.tmpl": {Data: []byte("package {{.Pkg}}\n")},
	}
	out := t.TempDir()
	err := WalkTemplateDest(src, struct{ Pkg string }{Pkg: "br.tec.lew.x"}, out, func(rel, destRel string) string {
		if rel == filepath.FromSlash("app/src/main/kotlin/Main.kt.tmpl") {
			return filepath.Join("app", "src", "main", "java", "br", "tec", "lew", "x", "Main.kt")
		}
		return destRel
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/x/Main.kt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "package br.tec.lew.x\n" {
		t.Fatalf("body: %q", got)
	}
}
