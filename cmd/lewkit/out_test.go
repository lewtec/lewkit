package main

import (
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/stretchr/testify/require"
)

func TestArtifactPathUsesFileInsideDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := artifactPath("android", dir, build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "contapila-debug.apk"), got)
}

func TestArtifactPathMissingDirectory(t *testing.T) {
	got, err := artifactPath("android", "dist", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, filepath.Join("dist", "contapila-debug.apk"), got)
}

func TestArtifactPathKeepsExplicitFile(t *testing.T) {
	got, err := artifactPath("android", "out/app.apk", build.Spec{Dir: t.TempDir(), ID: "br.tec.lew.contapila"})
	require.NoError(t, err)
	require.Equal(t, "out/app.apk", got)
}
