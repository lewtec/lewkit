//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// ErrUnavailable means WebView2Loader.dll is missing.
var ErrUnavailable = errors.New("webview2 unavailable")

var (
	loaderDLL          *syscall.LazyDLL
	createEnvironment  *syscall.LazyProc
	coInitializeEx     = syscall.NewLazyDLL("ole32.dll").NewProc("CoInitializeEx")
	coTaskMemFree      = syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree")
	createMemoryStream = syscall.NewLazyDLL("shlwapi.dll").NewProc("SHCreateMemStream")
)

// Available opens WebView2Loader.dll. WEBVIEW2_LOADER, when set, is a full path.
func Available() error {
	if loaderDLL != nil {
		return nil
	}
	name := os.Getenv("WEBVIEW2_LOADER")
	if name == "" {
		name = "WebView2Loader.dll"
	}
	dll := syscall.NewLazyDLL(name)
	if err := dll.Load(); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, name, err)
	}
	proc := dll.NewProc("CreateCoreWebView2EnvironmentWithOptions")
	if err := proc.Find(); err != nil {
		return fmt.Errorf("%w: CreateCoreWebView2EnvironmentWithOptions: %v", ErrUnavailable, err)
	}
	loaderDLL = dll
	createEnvironment = proc
	return nil
}

// CoInitialize enters the single-threaded apartment.
func CoInitialize() error {
	hr, _, _ := coInitializeEx.Call(0, 2)
	if int32(hr) < 0 && uint32(hr) != 0x00000001 {
		return fmt.Errorf("CoInitializeEx: %x", uint32(hr))
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
		return 0, fmt.Errorf("SHCreateMemStream: %v", err)
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
	hr, _, _ := createEnvironment.Call(0, uintptr(unsafe.Pointer(userDataFolder)), 0, handler)
	if int32(hr) < 0 {
		return fmt.Errorf("CreateCoreWebView2EnvironmentWithOptions: %x", uint32(hr))
	}
	return nil
}
