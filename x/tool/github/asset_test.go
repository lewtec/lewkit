package github

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHeliumAssetNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		os   string
		arch string
		ok   bool
	}{
		{"helium_0.19.2.1_arm64-macos.dmg", "darwin", "arm64", true},
		{"helium_0.19.2.1_x86_64-macos.dmg", "darwin", "amd64", true},
		{"helium-0.19.3.1-x86_64_linux.tar.xz", "linux", "amd64", true},
		{"helium-0.19.3.1-arm64_linux.tar.xz", "linux", "arm64", true},
		{"helium_0.19.2.1_x64-windows.zip", "windows", "amd64", true},
		{"sops-v3.10.2.exe", "windows", "amd64", true},
		{"helium_0.19.2.1_arm64-windows.zip", "windows", "arm64", true},
		{"sops-v3.10.2.amd64.exe", "windows", "amd64", true},
		{"sops-v3.10.2.arm64.exe", "windows", "arm64", true},
		{"sops-v3.10.2.amd64.exe.spdx.sbom.json", "", "", false},
		{"sops-v3.10.2.linux.amd64", "linux", "amd64", true},
		{"0.17.2.1-arm64.delta", "", "", false},
		{"helium-0.19.3.1-x86_64_linux.tar.xz.asc", "linux", "amd64", true},
	}
	for _, tc := range cases {
		osName, arch, ok := parseAssetName(tc.name)
		require.Equal(t, tc.ok, ok, tc.name)
		require.Equal(t, tc.os, osName, tc.name)
		require.Equal(t, tc.arch, arch, tc.name)
	}
}
