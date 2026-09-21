package tool

import (
	"path/filepath"
	"testing"
)

func TestSelectCodexCLIAsset(t *testing.T) {
	names := []string{
		"codex-x86_64-unknown-linux-musl.tar.gz",
		"codex-x86_64-unknown-linux-musl.zst",
		"codex-app-server-x86_64-unknown-linux-musl.tar.gz",
		"codex-package-x86_64-unknown-linux-musl.tar.gz",
		"codex-npm-linux-x64-0.142.5.tgz",
		"codex-zsh-x86_64-unknown-linux-musl.tar.gz",
		"codex-responses-api-proxy-x86_64-unknown-linux-musl.tar.gz",
	}
	artifacts := make([]Artifact, 0, len(names))
	for _, name := range names {
		artifacts = append(artifacts, Artifact{
			OS: "linux", Arch: "amd64",
			URL: "https://github.com/openai/codex/releases/download/rust-v0.142.5/" + name,
		})
	}
	got := SelectArtifact(artifacts, "linux", "amd64", "codex")
	if got == nil {
		t.Fatal("no artifact selected")
	}
	if base := filepath.Base(got.URL); base != "codex-x86_64-unknown-linux-musl.tar.gz" {
		t.Fatalf("SelectArtifact() = %s, want codex-x86_64-unknown-linux-musl.tar.gz", base)
	}
}

func TestSelectArtifactPrefersAndroidOverLinux(t *testing.T) {
	artifacts := []Artifact{
		{OS: "linux", Arch: "arm64", URL: "https://example.com/workspaced_Linux_arm64.tar.gz"},
		{OS: "android", Arch: "arm64", URL: "https://example.com/workspaced_Android_arm64.tar.gz"},
		{OS: "darwin", Arch: "arm64", URL: "https://example.com/workspaced_Darwin_arm64.tar.gz"},
	}
	got := SelectArtifact(artifacts, "android", "arm64", "workspaced")
	if got == nil {
		t.Fatal("no artifact selected")
	}
	if got.OS != "android" {
		t.Fatalf("SelectArtifact() OS = %s, want android (URL %s)", got.OS, got.URL)
	}
}

func TestSelectArtifactAndroidFallsBackToLinux(t *testing.T) {
	artifacts := []Artifact{
		{OS: "linux", Arch: "arm64", URL: "https://example.com/tool_Linux_arm64.tar.gz"},
		{OS: "darwin", Arch: "arm64", URL: "https://example.com/tool_Darwin_arm64.tar.gz"},
	}
	got := SelectArtifact(artifacts, "android", "arm64", "tool")
	if got == nil {
		t.Fatal("no artifact selected")
	}
	if got.OS != "linux" {
		t.Fatalf("SelectArtifact() OS = %s, want linux fallback", got.OS)
	}
}
