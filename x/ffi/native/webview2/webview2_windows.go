//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means the installed Edge WebView2 runtime could not be opened.
var ErrUnavailable = errors.New("webview2 unavailable")

const (
	hkeyCurrentUser  = 0x80000001
	hkeyLocalMachine = 0x80000002
	keyRead          = 0x20019
	keyWow64_32      = 0x0200
	regSZ            = 1
	regExpandSZ      = 2

	clientStateKey = `Software\Microsoft\EdgeUpdate\ClientState\`
)

var (
	createEnvironment  native.Proc
	regOpen            = native.ProcOf("advapi32.dll", "RegOpenKeyExW")
	regQuery           = native.ProcOf("advapi32.dll", "RegQueryValueExW")
	regClose           = native.ProcOf("advapi32.dll", "RegCloseKey")
	expandEnv          = native.ProcOf("kernel32.dll", "ExpandEnvironmentStringsW")
	coInitializeEx     = native.ProcOf("ole32.dll", "CoInitializeEx")
	coTaskMemFree      = native.ProcOf("ole32.dll", "CoTaskMemFree")
	createMemoryStream = native.ProcOf("shlwapi.dll", "SHCreateMemStream")

	// User32 and kernel32 entry points for the window the runtime attaches to.
	RegisterClassEx  = native.ProcOf("user32.dll", "RegisterClassExW")
	CreateWindowEx   = native.ProcOf("user32.dll", "CreateWindowExW")
	DefWindowProc    = native.ProcOf("user32.dll", "DefWindowProcW")
	GetMessage       = native.ProcOf("user32.dll", "GetMessageW")
	PeekMessage      = native.ProcOf("user32.dll", "PeekMessageW")
	TranslateMessage = native.ProcOf("user32.dll", "TranslateMessage")
	DispatchMessage  = native.ProcOf("user32.dll", "DispatchMessageW")
	ShowWindow       = native.ProcOf("user32.dll", "ShowWindow")
	DestroyWindow    = native.ProcOf("user32.dll", "DestroyWindow")
	PostMessage      = native.ProcOf("user32.dll", "PostMessageW")
	GetClientRect    = native.ProcOf("user32.dll", "GetClientRect")
	GetModuleHandle  = native.ProcOf("kernel32.dll", "GetModuleHandleW")
)

// Available opens EmbeddedBrowserWebView.dll from the installed Edge WebView2 runtime.
var available = native.Once(func() error {
	path, err := findClient(readInstall, archName(runtime.GOARCH), fileExists)
	if err != nil {
		return err
	}
	createEnvironment = native.ProcOf(path, "CreateWebViewEnvironmentWithOptionsInternal")
	if err := createEnvironment.Find(); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, path, err)
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
	// CreateWebViewEnvironmentWithOptionsInternal(checkRunningInstance, installed, userData, options, handler).
	hr, _, _ := createEnvironment.Call(1, 0, uintptr(unsafe.Pointer(userDataFolder)), 0, handler)
	if int32(hr) < 0 {
		return fmt.Errorf("CreateWebViewEnvironmentWithOptionsInternal: %x", uint32(hr))
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func readInstall(hive, guid string) string {
	root := uintptr(hkeyCurrentUser)
	if hive == hiveLocalMachine {
		root = hkeyLocalMachine
	}
	folder, err := queryString(root, clientStateKey+guid, "EBWebView")
	if err != nil {
		return ""
	}
	return folder
}

func queryString(root uintptr, subkey, value string) (string, error) {
	sub, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return "", err
	}
	var key syscall.Handle
	status, _, _ := regOpen.Call(root, uintptr(unsafe.Pointer(sub)), 0, keyRead|keyWow64_32, uintptr(unsafe.Pointer(&key)))
	if status != 0 {
		return "", syscall.Errno(status)
	}
	defer regClose.Call(uintptr(key))

	name, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return "", err
	}
	buf := make([]uint16, 4096)
	kind := uint32(0)
	size := uint32(len(buf) * 2)
	status, _, _ = regQuery.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(name)),
		0,
		uintptr(unsafe.Pointer(&kind)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if status != 0 {
		return "", syscall.Errno(status)
	}
	n := int(size / 2)
	if n > len(buf) {
		n = len(buf)
	}
	if n == 0 {
		return "", nil
	}
	text := syscall.UTF16ToString(buf[:n])
	if kind == regExpandSZ {
		return expandString(text), nil
	}
	if kind != regSZ {
		return "", fmt.Errorf("registry type %d", kind)
	}
	return text, nil
}

func expandString(text string) string {
	in, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return text
	}
	size := len(text) + 1
	for range 2 {
		buf := make([]uint16, size)
		n, _, _ := expandEnv.Call(uintptr(unsafe.Pointer(in)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			return text
		}
		if int(n) <= len(buf) {
			return syscall.UTF16ToString(buf[:n])
		}
		size = int(n)
	}
	return text
}
