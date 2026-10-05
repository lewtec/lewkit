//go:build windows

package vulkan

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	hkeyCurrentUser  = 0x80000001
	hkeyLocalMachine = 0x80000002
	keyRead          = 0x20019
	keyWow64_64      = 0x0100
	keyWow64_32      = 0x0200
	regSZ            = 1
	regExpandSZ      = 2
	regMoreData      = 234
	regNoMoreItems   = 259
	loadDLLDir       = 0x00000100
	loadDefaultDirs  = 0x00001000
	loadAltered      = 0x00000008
)

var (
	regOpen            = native.ProcOf("advapi32.dll", "RegOpenKeyExW")
	regQuery           = native.ProcOf("advapi32.dll", "RegQueryValueExW")
	regEnumValue       = native.ProcOf("advapi32.dll", "RegEnumValueW")
	regEnumKey         = native.ProcOf("advapi32.dll", "RegEnumKeyExW")
	regClose           = native.ProcOf("advapi32.dll", "RegCloseKey")
	expandEnv          = native.ProcOf("kernel32.dll", "ExpandEnvironmentStringsW")
	loadLibraryEx      = native.ProcOf("kernel32.dll", "LoadLibraryExW")
	getSystemDirectory = native.ProcOf("kernel32.dll", "GetSystemDirectoryW")
	searchPathW        = native.ProcOf("kernel32.dll", "SearchPathW")
)

var khronosKeys = []string{
	`SOFTWARE\Khronos\Vulkan\Drivers`,
	`SOFTWARE\Khronos\Vulkan\ImplicitLayers`,
	`SOFTWARE\Khronos\Vulkan\ExplicitLayers`,
}

func windowsLibNames() []string {
	sdks := []string{
		os.Getenv("VULKAN_SDK"),
		os.Getenv("VK_SDK_PATH"),
	}
	for _, root := range []uintptr{hkeyCurrentUser, hkeyLocalMachine} {
		sub := `Environment`
		if root == hkeyLocalMachine {
			sub = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
		}
		for _, view := range []uintptr{keyWow64_64, keyWow64_32} {
			sdks = append(sdks,
				registryString(root, sub, `VULKAN_SDK`, view),
				registryString(root, sub, `VK_SDK_PATH`, view),
			)
		}
	}
	sdks = append(sdks, uninstallRoots()...)
	sdks = append(sdks, defaultSDKDirs()...)
	var manifests []string
	for _, view := range []uintptr{keyWow64_64, keyWow64_32} {
		for _, root := range []uintptr{hkeyLocalMachine, hkeyCurrentUser} {
			for _, sub := range khronosKeys {
				for _, name := range enumValueNames(root, sub, view) {
					manifests = append(manifests, manifestPath(root, sub, view, name)...)
				}
			}
		}
	}
	var extra []string
	sys := systemDirectory()
	if sys == "" {
		if root := os.Getenv("SystemRoot"); root != "" {
			sys = filepath.Join(root, "System32")
		}
	}
	for _, name := range loaderFileNames(runtime.GOARCH) {
		if sys != "" {
			extra = append(extra, filepath.Join(sys, name))
		}
		if found := searchedDLL(name); found != "" {
			extra = append(extra, found)
		}
	}
	// The runtime installer leaves vulkan-1-<version>.dll in System32 even
	// when the vulkan-1.dll copy is missing. SDK folders can hold the same.
	if sys != "" {
		extra = append(extra, versionedLoaders(sys)...)
	}
	for _, sdk := range sdks {
		extra = append(extra, versionedLoaders(sdk)...)
		extra = append(extra, versionedLoaders(filepath.Join(sdk, "Bin"))...)
	}
	var drivers []string
	for _, manifest := range manifests {
		drivers = append(drivers, manifestLoaders(manifest)...)
	}
	drivers = append(drivers, driverStoreLoaders()...)
	return loaderCandidates(os.Getenv("SystemRoot"), runtime.GOARCH, extra, sdks, manifests, drivers)
}

// manifestPath accepts the Khronos value name when it is a file path.
// Some installs store that path in the value data instead.
func manifestPath(root uintptr, sub string, view uintptr, name string) []string {
	if name == "" {
		return nil
	}
	if strings.ContainsAny(name, `\/`) {
		return []string{name}
	}
	if data := registryString(root, sub, name, view); strings.ContainsAny(data, `\/`) {
		return []string{data}
	}
	return nil
}

// manifestLoaders reads an ICD or layer JSON and returns vulkan-1.dll
// beside that file's library_path, and in the parent directory.
// The driver store keeps the loader next to the driver, outside System32.
func manifestLoaders(path string) []string {
	if path == "" {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	buf := make([]byte, 1<<20)
	n, _ := file.Read(buf)
	if n == 0 {
		return nil
	}
	loader := besideLoader(path, quotedLibrary(string(buf[:n])))
	dir := winDir(loader)
	if dir == "" {
		return nil
	}
	var out []string
	for _, nameDir := range []string{dir, winDir(dir)} {
		for _, name := range loaderFileNames(runtime.GOARCH) {
			if nameDir == "" {
				continue
			}
			out = append(out, nameDir+`\`+name)
		}
	}
	return out
}

// driverStoreLoaders finds a staged loader inside a driver package.
// NVIDIA, Intel, and AMD each use their own filename. AMD often nests
// it one directory down (Bxxxxxx). System32's vulkan-1.dll is only the
// copy made when that install step runs.
func driverStoreLoaders() []string {
	sys := systemDirectory()
	if sys == "" {
		if root := os.Getenv("SystemRoot"); root != "" {
			sys = filepath.Join(root, "System32")
		}
	}
	if sys == "" {
		return nil
	}
	root := filepath.Join(sys, "DriverStore", "FileRepository")
	packages, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	names := loaderFileNames(runtime.GOARCH)
	var found []string
	for _, pkg := range packages {
		if !pkg.IsDir() {
			continue
		}
		dir := filepath.Join(root, pkg.Name())
		found = append(found, loadersIn(dir, names)...)
		nested, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range nested {
			if !entry.IsDir() {
				continue
			}
			found = append(found, loadersIn(filepath.Join(dir, entry.Name()), names)...)
		}
	}
	return found
}

func loadersIn(dir string, names []string) []string {
	var found []string
	for _, name := range names {
		if loader := existingDLL(filepath.Join(dir, name)); loader != "" {
			found = append(found, loader)
		}
	}
	return found
}

func versionedLoaders(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !versionedLoader(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	names = preferVersioned(names)
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = filepath.Join(dir, name)
	}
	return out
}

func existingDLL(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	return path
}

func registryString(root uintptr, subkey, value string, view uintptr) string {
	key, ok := openKey(root, subkey, view)
	if !ok {
		return ""
	}
	defer regClose.Call(key)
	name, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 1024)
	kind := uint32(0)
	size := uint32(len(buf) * 2)
	status, _, _ := regQuery.Call(
		key,
		uintptr(unsafe.Pointer(name)),
		0,
		uintptr(unsafe.Pointer(&kind)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if status != 0 {
		return ""
	}
	n := int(size / 2)
	if n > len(buf) {
		n = len(buf)
	}
	if n == 0 {
		return ""
	}
	text := syscall.UTF16ToString(buf[:n])
	if kind == regExpandSZ {
		return expandString(text)
	}
	if kind != regSZ {
		return ""
	}
	return text
}

func enumValueNames(root uintptr, subkey string, view uintptr) []string {
	key, ok := openKey(root, subkey, view)
	if !ok {
		return nil
	}
	defer regClose.Call(key)
	buf := make([]uint16, 32768)
	var names []string
	for i := uintptr(0); ; i++ {
		n := uint32(len(buf))
		status, _, _ := regEnumValue.Call(
			key,
			i,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&n)),
			0, 0, 0, 0,
		)
		if status == regMoreData {
			buf = make([]uint16, len(buf)*2)
			i--
			continue
		}
		if status == regNoMoreItems || status != 0 {
			break
		}
		if n > uint32(len(buf)) {
			n = uint32(len(buf))
		}
		names = append(names, syscall.UTF16ToString(buf[:n]))
	}
	return names
}

func uninstallRoots() []string {
	bases := []struct {
		root uintptr
		path string
	}{
		{hkeyLocalMachine, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{hkeyLocalMachine, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{hkeyCurrentUser, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{hkeyCurrentUser, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}
	var roots []string
	for _, base := range bases {
		views := []uintptr{keyWow64_64}
		if !strings.Contains(base.path, "WOW6432Node") {
			views = append(views, keyWow64_32)
		}
		for _, view := range views {
			for _, name := range enumSubkeys(base.root, base.path, view) {
				sub := base.path + `\` + name
				root := sdkRoot(
					registryString(base.root, sub, "DisplayName", view),
					registryString(base.root, sub, "InstallLocation", view),
					registryString(base.root, sub, "UninstallString", view),
				)
				if root != "" {
					roots = append(roots, root)
				}
			}
		}
	}
	return roots
}

func enumSubkeys(root uintptr, subkey string, view uintptr) []string {
	key, ok := openKey(root, subkey, view)
	if !ok {
		return nil
	}
	defer regClose.Call(key)
	buf := make([]uint16, 256)
	var names []string
	for i := uintptr(0); ; i++ {
		n := uint32(len(buf))
		status, _, _ := regEnumKey.Call(
			key,
			i,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&n)),
			0, 0, 0, 0,
		)
		if status == regMoreData {
			buf = make([]uint16, len(buf)*2)
			i--
			continue
		}
		if status != 0 {
			break
		}
		if n > uint32(len(buf)) {
			n = uint32(len(buf))
		}
		names = append(names, syscall.UTF16ToString(buf[:n]))
	}
	return names
}

func defaultSDKDirs() []string {
	drive := os.Getenv("SystemDrive")
	if drive == "" {
		drive = `C:`
	}
	root := drive + `\VulkanSDK`
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := []string{root}
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, filepath.Join(root, entry.Name()))
		}
	}
	return out
}

func systemDirectory() string {
	buf := make([]uint16, 260)
	n, _, _ := getSystemDirectory.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || int(n) >= len(buf) {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func searchedDLL(name string) string {
	file, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 32768)
	n, _, _ := searchPathW.Call(0, uintptr(unsafe.Pointer(file)), 0, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])), 0)
	if n == 0 || int(n) >= len(buf) {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func openKey(root uintptr, subkey string, view uintptr) (uintptr, bool) {
	sub, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return 0, false
	}
	var key syscall.Handle
	status, _, _ := regOpen.Call(root, uintptr(unsafe.Pointer(sub)), 0, keyRead|view, uintptr(unsafe.Pointer(&key)))
	if status != 0 {
		return 0, false
	}
	return uintptr(key), true
}

var (
	loaderOnce   sync.Once
	loaderHandle uintptr
	loaderErr    error
)

func openLoader() (uintptr, error) {
	loaderOnce.Do(func() {
		names := libNames()
		var last error
		for _, name := range names {
			handle, err := loadVulkanDLL(name)
			if err != nil {
				last = err
				continue
			}
			// An ICD exports vk_icdGetInstanceProcAddr, not the loader entry.
			if _, symErr := native.Symbol(handle, "vkGetInstanceProcAddr"); symErr != nil {
				last = fmt.Errorf("load %s: vkGetInstanceProcAddr: %w", name, symErr)
				continue
			}
			loaderHandle = handle
			return
		}
		if last == nil {
			last = fmt.Errorf("vulkan-1.dll")
		}
		loaderErr = fmt.Errorf("%w (tried %s)", last, strings.Join(names, ", "))
	})
	return loaderHandle, loaderErr
}

func loadVulkanDLL(path string) (uintptr, error) {
	// LoadDLL keeps the UTF-16 path alive. A bare LoadLibraryExW call does not.
	dll, err := syscall.LoadDLL(path)
	if err == nil {
		return uintptr(dll.Handle), nil
	}
	if !filepath.IsAbs(path) {
		return 0, err
	}
	// A driver-store loader depends on DLLs sitting in that same directory.
	file, convErr := syscall.UTF16PtrFromString(path)
	if convErr != nil {
		return 0, convErr
	}
	for _, flag := range []uintptr{loadAltered, loadDLLDir | loadDefaultDirs} {
		handle, _, callErr := loadLibraryEx.Call(uintptr(unsafe.Pointer(file)), 0, flag)
		runtime.KeepAlive(file)
		if handle != 0 {
			return handle, nil
		}
		if callErr != nil {
			err = callErr
		}
	}
	return 0, fmt.Errorf("load %s: %w", path, err)
}

func expandString(text string) string {
	in, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return text
	}
	size, _, _ := expandEnv.Call(uintptr(unsafe.Pointer(in)), 0, 0)
	if size == 0 {
		return text
	}
	buf := make([]uint16, size)
	n, _, _ := expandEnv.Call(uintptr(unsafe.Pointer(in)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return text
	}
	return syscall.UTF16ToString(buf)
}
