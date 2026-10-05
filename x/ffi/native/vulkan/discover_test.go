package vulkan

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoaderCandidates(t *testing.T) {
	root := filepath.Join("C:", "Windows")
	sdk := filepath.Join("C:", "VulkanSDK", "1.3.280.0")
	driver := filepath.Join(root, "System32", "DriverStore", "FileRepository", "nv.inf_amd64", "vulkan-1.dll")
	got := loaderCandidates(root, "amd64", []string{filepath.Join(root, "System32", "vulkan-1.dll")}, []string{sdk, sdk}, []string{
		filepath.Join(sdk, "Bin", "VkICD_mock_icd.json"),
		filepath.Join(root, "System32", "nv-vk64.json"),
		"",
	}, []string{driver, driver})
	require.Equal(t, "vulkan-1.dll", got[0])
	require.Contains(t, got, "vulkan-1-x64.dll")
	require.Contains(t, got, "vulkan-1-64.dll")
	require.Contains(t, got, "vulkan64.dll")
	require.Contains(t, got, filepath.Join(root, "System32", "vulkan-1.dll"))
	require.Contains(t, got, filepath.Join(root, "System32", "vulkan-1-x64.dll"))
	require.Contains(t, got, filepath.Join(root, "System32", "vulkan64.dll"))
	require.Contains(t, got, filepath.Join(sdk, "vulkan-1.dll"))
	require.Contains(t, got, filepath.Join(sdk, "X64", "vulkan-1.dll"))
	require.Contains(t, got, filepath.Join(sdk, "Bin", "vulkan-1.dll"))
	require.Contains(t, got, filepath.Join(sdk, "RunTimeInstaller", "X64", "vulkan-1.dll"))
	require.Contains(t, got, filepath.Join(sdk, "RunTimeInstaller", "x64", "vulkan-1.dll"))
	require.Contains(t, got, driver)
	seen := map[string]int{}
	for _, path := range got {
		seen[path]++
	}
	for path, n := range seen {
		require.Equal(t, 1, n, path)
	}
	require.Equal(t, []string{"vulkan-1.dll", "vulkan-1-x64.dll", "vulkan-1-64.dll", "vulkan64.dll"}, loaderCandidates("", "", nil, nil, nil, nil))
	arm := loaderCandidates("", "arm64", nil, []string{sdk}, nil, nil)
	require.Contains(t, arm, filepath.Join(sdk, "RunTimeInstaller", "ARM64", "vulkan-1.dll"))
}

func TestLoaderFileNames(t *testing.T) {
	require.Equal(t, []string{"vulkan-1.dll", "vulkan-1-x64.dll", "vulkan-1-64.dll", "vulkan64.dll"}, loaderFileNames("amd64"))
	require.Equal(t, []string{"vulkan-1.dll", "vulkan-1-x86.dll", "vulkan-1-32.dll", "vulkan32.dll"}, loaderFileNames("386"))
	require.Equal(t, []string{"vulkan-1.dll", "vulkan-1-a64.dll", "vulkan-1-a64ec.dll"}, loaderFileNames("arm64"))
}

func TestVersionedLoader(t *testing.T) {
	require.True(t, versionedLoader("vulkan-1-1-3-280-0.dll"))
	require.True(t, versionedLoader(`C:\Windows\System32\vulkan-1-2-0-1-1.dll`))
	require.True(t, versionedLoader("vulkan-1-999-0-0-0.dll"))
	require.False(t, versionedLoader("vulkan-1.dll"))
	require.False(t, versionedLoader("vulkan-1-x64.dll"))
	require.False(t, versionedLoader("vulkan64.dll"))
	require.False(t, versionedLoader("vulkan-1-1-3-280-0.exe"))
	require.False(t, versionedLoader("nvoglv64.dll"))
	got := preferVersioned([]string{
		"vulkan-1-999-0-0-0.dll",
		"vulkan-1-1-3-280-0.dll",
		"vulkan-1-1-3-90-0.dll",
		"vulkan-1-x64.dll",
		"vulkan-1-1-2-0-1-1.dll",
	})
	require.Equal(t, []string{
		"vulkan-1-1-3-280-0.dll",
		"vulkan-1-1-3-90-0.dll",
		"vulkan-1-999-0-0-0.dll",
	}, got)
}

func TestBesideLoader(t *testing.T) {
	manifest := `C:\Windows\System32\nv-vk64.json`
	body := `{
		"file_format_version": "1.0.0",
		"ICD": {
			"library_path": "C:\\Windows\\System32\\DriverStore\\FileRepository\\nv_disp.inf_amd64_abc\\nvoglv64.dll",
			"api_version": "1.3.280"
		}
	}`
	library := quotedLibrary(body)
	require.Equal(t, `C:\Windows\System32\DriverStore\FileRepository\nv_disp.inf_amd64_abc\nvoglv64.dll`, library)
	require.Equal(t,
		`C:\Windows\System32\DriverStore\FileRepository\nv_disp.inf_amd64_abc\vulkan-1.dll`,
		besideLoader(manifest, library),
	)
	require.Equal(t, `C:\Windows\System32\vulkan-1.dll`, besideLoader(manifest, `.\nvoglv64.dll`))
	require.Equal(t, "", besideLoader("", `nvoglv64.dll`))
	require.Equal(t, "", quotedLibrary(`{"ICD":{}}`))
}

func TestSDKRoot(t *testing.T) {
	require.Equal(t, "", sdkRoot("Edge WebView", `C:\Edge`, ""))
	require.Equal(t, `C:\VulkanSDK\1.3.280.0`, sdkRoot("Vulkan SDK 1.3.280.0", `C:\VulkanSDK\1.3.280.0\`, ""))
	require.Equal(t, `C:\VulkanSDK\1.3.280.0`, sdkRoot(
		"Vulkan Runtime",
		"",
		`"C:\VulkanSDK\1.3.280.0\maintenancetool.exe" --uninstall`,
	))
}
