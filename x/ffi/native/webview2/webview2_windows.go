//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
	"golang.org/x/sys/windows/registry"
)

// ErrUnavailable means the Edge WebView2 runtime is not installed.
var ErrUnavailable = errors.New("webview2 unavailable")

var errCOM = errors.New("webview com call failed")

var (
	createEnvironment native.Proc
	runtimeInternal   bool

	coInitializeEx     = native.ProcOf("ole32.dll", "CoInitializeEx")
	coTaskMemFree      = native.ProcOf("ole32.dll", "CoTaskMemFree")
	createMemoryStream = native.ProcOf("shlwapi.dll", "SHCreateMemStream")
)

// Available opens the WebView2 runtime that Windows already has.
// WEBVIEW2_LOADER, when set, is a full path to WebView2Loader.dll and wins.
// A loader already on the machine is next. Otherwise this loads
// EmbeddedBrowserWebView.dll from the installed Edge runtime.
var available = native.Once(func() error {
	if name := os.Getenv("WEBVIEW2_LOADER"); name != "" {
		return bindLoader(name)
	}
	loader := native.ProcOf("WebView2Loader.dll", "CreateCoreWebView2EnvironmentWithOptions")
	if err := loader.Find(); err == nil {
		createEnvironment = loader
		return nil
	}
	dll, err := installedRuntimeDLL()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	createEnvironment = native.ProcOf(dll, "CreateWebViewEnvironmentWithOptionsInternal")
	if err := createEnvironment.Find(); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnavailable, dll, err)
	}
	runtimeInternal = true
	return nil
})

func bindLoader(name string) error {
	createEnvironment = native.ProcOf(name, "CreateCoreWebView2EnvironmentWithOptions")
	if err := createEnvironment.Find(); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnavailable, name, err)
	}
	return nil
}

func Available() error { return available() }

// CoInitialize enters the single-threaded apartment.
func CoInitialize() error {
	hr, _, _ := coInitializeEx.Call(0, 2)
	if int32(hr) < 0 && uint32(hr) != 0x00000001 {
		return fmt.Errorf("%w: CoInitializeEx %x", errCOM, uint32(hr))
	}
	return nil
}

// FreeTaskMemory releases a string returned by WebView2.
func FreeTaskMemory(pointer uintptr) {
	if pointer != 0 {
		_, _, _ = coTaskMemFree.Call(pointer)
	}
}

// MemoryStream returns an IStream over body. The caller Releases it.
func MemoryStream(body []byte) (uintptr, error) {
	var data uintptr
	if len(body) > 0 {
		data = uintptr(unsafe.Pointer(&body[0]))
	}
	stream, _, err := createMemoryStream.Call(data, uintptr(len(body)))
	if stream == 0 {
		if err == nil {
			err = errCOM
		}
		return 0, fmt.Errorf("%w: SHCreateMemStream", err)
	}
	return stream, nil
}

// CreateEnvironment starts the WebView2 runtime. handler is
// ICoreWebView2CreateCoreWebView2EnvironmentCompletedHandler.
// userDataFolder is the profile directory.
func CreateEnvironment(userDataFolder *uint16, handler uintptr) error {
	if err := Available(); err != nil {
		return err
	}
	var hr uintptr
	call := "CreateCoreWebView2EnvironmentWithOptions"
	if runtimeInternal {
		// checkRunningInstance is 1. runtimeType 0 is the installed runtime.
		// Options stay null, the same as the public loader call.
		hr, _, _ = createEnvironment.Call(1, 0, uintptr(unsafe.Pointer(userDataFolder)), 0, handler)
		call = "CreateWebViewEnvironmentWithOptionsInternal"
	} else {
		hr, _, _ = createEnvironment.Call(0, uintptr(unsafe.Pointer(userDataFolder)), 0, handler)
	}
	if int32(hr) < 0 {
		return fmt.Errorf("%w: %s %x", errCOM, call, uint32(hr))
	}
	return nil
}

// installedRuntimeDLL finds EmbeddedBrowserWebView.dll from EdgeUpdate.
// ClientState\EBWebView is the version folder. Clients\pv is the version,
// joined onto the usual per-machine and per-user install directories.
func installedRuntimeDLL() (string, error) {
	arch := webViewArch(runtime.GOARCH)
	if arch == "" {
		return "", fmt.Errorf("%w: %s", errNoRuntime, runtime.GOARCH)
	}
	for _, guid := range channelGUID {
		for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
			if folder, err := regString(root, clientStatePrefix+guid, "EBWebView"); err == nil && folder != "" {
				if path, err := acceptInstall(folder, arch); err == nil {
					return path, nil
				}
			}
			ver, err := regString(root, clientsPrefix+guid, "pv")
			if err != nil || !versionAtLeast(ver, minWebViewVersion) {
				continue
			}
			for _, base := range installRoots() {
				if path, err := acceptInstall(filepath.Join(base, ver), arch); err == nil {
					return path, nil
				}
			}
		}
	}
	return "", errNoRuntime
}

func regString(root registry.Key, path, name string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE|registry.WOW64_32KEY)
	if err != nil {
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	return value, err
}

func installRoots() []string {
	var roots []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		dir := os.Getenv(env)
		if dir == "" {
			continue
		}
		roots = append(roots, filepath.Join(dir, "Microsoft", "EdgeWebView", "Application"))
	}
	return roots
}
