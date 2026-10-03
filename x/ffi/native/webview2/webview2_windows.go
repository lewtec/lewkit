//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means WebView2Loader.dll is missing.
var ErrUnavailable = errors.New("webview2 unavailable")

var (
	createEnvironment  native.Proc
	coInitializeEx     = native.ProcOf("ole32.dll", "CoInitializeEx")
	coTaskMemFree      = native.ProcOf("ole32.dll", "CoTaskMemFree")
	createMemoryStream = native.ProcOf("shlwapi.dll", "SHCreateMemStream")

	// User32 and kernel32 entry points for the window the loader attaches to.
	RegisterClassEx  = native.ProcOf("user32.dll", "RegisterClassExW")
	CreateWindowEx   = native.ProcOf("user32.dll", "CreateWindowExW")
	DefWindowProc    = native.ProcOf("user32.dll", "DefWindowProcW")
	GetMessage       = native.ProcOf("user32.dll", "GetMessageW")
	TranslateMessage = native.ProcOf("user32.dll", "TranslateMessage")
	DispatchMessage  = native.ProcOf("user32.dll", "DispatchMessageW")
	ShowWindow       = native.ProcOf("user32.dll", "ShowWindow")
	DestroyWindow    = native.ProcOf("user32.dll", "DestroyWindow")
	PostMessage      = native.ProcOf("user32.dll", "PostMessageW")
	GetClientRect    = native.ProcOf("user32.dll", "GetClientRect")
	GetModuleHandle  = native.ProcOf("kernel32.dll", "GetModuleHandleW")
)

// Available opens WebView2Loader.dll. WEBVIEW2_LOADER, when set, is a full path.
var available = native.Once(func() error {
	name := os.Getenv("WEBVIEW2_LOADER")
	if name == "" {
		name = "WebView2Loader.dll"
	}
	createEnvironment = native.ProcOf(name, "CreateCoreWebView2EnvironmentWithOptions")
	if err := createEnvironment.Find(); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, name, err)
	}
	return nil
})

func Available() error { return available() }

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
