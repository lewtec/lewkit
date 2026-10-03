//go:build windows

package win32

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	clsctxInprocServer = 1
	rpcEChangedMode    = 0x80010106
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	cwUseDefault       = 0x80000000
	swShowNormal       = 1
	sigdnFileSysPath   = 0x80058000
	fosOverwrite       = 0x2
	fosNoChangeDir     = 0x8
	fosPickFolders     = 0x20
	fosForceFileSystem = 0x40
	fosAllowMulti      = 0x200
	fosPathMustExist   = 0x800
	fosFileMustExist   = 0x1000
	hrCanceled         = 0x800704C7
	eFail              = 0x80004005
	pmNoRemove         = 0

	// IFileDialog vtable. IUnknown is 0..2, IModalWindow.Show is 3,
	// then IFileDialog in IDL order. IFileOpenDialog.GetResults is 27.
	slotShow         = 3
	slotSetFileTypes = 4
	slotSetOptions   = 9
	slotGetOptions   = 10
	slotSetFolder    = 12
	slotSetFileName  = 15
	slotSetTitle     = 17
	slotGetResult    = 20
	slotGetResults   = 27
)

var errDialog = errors.New("file dialog")

var (
	clsidOpen = syscall.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	clsidSave = syscall.GUID{Data1: 0xC0B4E2F3, Data2: 0xBA21, Data3: 0x4773, Data4: [8]byte{0x8D, 0xBA, 0x33, 0x5E, 0xC9, 0x46, 0xEB, 0x8B}}
	iidOpen   = syscall.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidSave   = syscall.GUID{Data1: 0x84BCCD23, Data2: 0x5FDE, Data3: 0x4CDB, Data4: [8]byte{0xAE, 0xA4, 0xAF, 0x64, 0xB8, 0x3D, 0x78, 0xAB}}
	iidItem   = syscall.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}

	procOleInitialize               = native.ProcOf("ole32.dll", "OleInitialize")
	procOleUninitialize             = native.ProcOf("ole32.dll", "OleUninitialize")
	procCoCreateInstance            = native.ProcOf("ole32.dll", "CoCreateInstance")
	procCoTaskMemFree               = native.ProcOf("ole32.dll", "CoTaskMemFree")
	procSHCreateItemFromParsingName = native.ProcOf("shell32.dll", "SHCreateItemFromParsingName")
	procRegisterClassExW            = native.ProcOf("user32.dll", "RegisterClassExW")
	procCreateWindowExW             = native.ProcOf("user32.dll", "CreateWindowExW")
	procDestroyWindow               = native.ProcOf("user32.dll", "DestroyWindow")
	procDefWindowProcW              = native.ProcOf("user32.dll", "DefWindowProcW")
	procShowWindow                  = native.ProcOf("user32.dll", "ShowWindow")
	procGetForegroundWindow         = native.ProcOf("user32.dll", "GetForegroundWindow")
	procGetWindowThreadProcessId    = native.ProcOf("user32.dll", "GetWindowThreadProcessId")
	procSetForegroundWindow         = native.ProcOf("user32.dll", "SetForegroundWindow")
	procBringWindowToTop            = native.ProcOf("user32.dll", "BringWindowToTop")
	procAttachThreadInput           = native.ProcOf("user32.dll", "AttachThreadInput")
	procAllowSetForegroundWindow    = native.ProcOf("user32.dll", "AllowSetForegroundWindow")
	procGetModuleHandleW            = native.ProcOf("kernel32.dll", "GetModuleHandleW")
	procGetCurrentThreadId          = native.ProcOf("kernel32.dll", "GetCurrentThreadId")
	procPeekMessageW                = native.ProcOf("user32.dll", "PeekMessageW")

	ownerOnce  sync.Once
	ownerName  = syscall.StringToUTF16Ptr("lewkit.filedialog.owner")
	ownerTitle = syscall.StringToUTF16Ptr("Open")
	ownerProc  = syscall.NewCallback(ownerWndProc)
)

type ownerClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

type filterSpec struct {
	name *uint16
	spec *uint16
}

func (opener) Choose(ctx context.Context, req filedialog.Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", filedialog.ErrRequest, err)
	}
	return showDedicated(req)
}

// queueMessage is a Windows MSG. Field alignment matches the system
// struct: 28 bytes on 386, 48 on amd64.
type queueMessage struct {
	hwnd    uintptr
	message uint32
	wparam  uintptr
	lparam  uintptr
	time    uint32
	ptX     int32
	ptY     int32
}

// dialogText is the UTF-16 Show still reads. SetTitle and SetFileTypes
// are not a reason to drop the strings before the dialog returns.
type dialogText struct {
	title *uint16
	name  *uint16
	text  [][]uint16
	specs []filterSpec
}

var (
	_ = [1]struct{}{}[(unsafe.Sizeof(queueMessage{})-28)*(unsafe.Sizeof(queueMessage{})-48)]
	_ = [1]struct{}{}[(unsafe.Sizeof(ownerClassEx{})-48)*(unsafe.Sizeof(ownerClassEx{})-80)]
)

func oleInit() error {
	hr, _, callErr := procOleInitialize.Call(0)
	if uint32(hr) == rpcEChangedMode || int32(hr) < 0 {
		return statusErr("com", hr, callErr)
	}
	return nil
}

func showDedicated(req filedialog.Request) ([]string, error) {
	type result struct {
		paths []string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		// A fresh STA. The WebView2 window is a different thread, and Show
		// on that thread returns E_FAIL. OleInitialize, not CoInitializeEx
		// with OLE1 DDE disabled: the shell dialog needs OLE.
		if err := oleInit(); err != nil {
			done <- result{err: err}
			return
		}
		defer procOleUninitialize.Call()
		var queued queueMessage
		procPeekMessageW.Call(uintptr(unsafe.Pointer(&queued)), 0, 0, 0, pmNoRemove)
		paths, err := showPrepared(req, 0)
		runtime.KeepAlive(&queued)
		done <- result{paths: paths, err: err}
	}()
	out := <-done
	return out.paths, out.err
}

func showPrepared(req filedialog.Request, owner uintptr) ([]string, error) {
	class, iid := &clsidOpen, &iidOpen
	if req.Save {
		class, iid = &clsidSave, &iidSave
	}
	var dialog uintptr
	hr, _, callErr := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(class)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if failed(hr) || dialog == 0 {
		return nil, statusErr("dialog", hr, callErr)
	}
	var temporary uintptr
	defer func() {
		release(dialog)
		if temporary != 0 {
			procDestroyWindow.Call(temporary)
		}
	}()

	kept, err := prepare(dialog, req)
	if err != nil {
		return nil, err
	}
	defer runtime.KeepAlive(&kept)
	if owner != 0 {
		hr, err = showDialog(dialog, owner)
	} else {
		hr, err, temporary = presentDialog(dialog)
	}
	if uint32(hr) == hrCanceled {
		return nil, filedialog.ErrCanceled
	}
	if err != nil {
		return nil, err
	}
	if req.Multiple {
		return collectMany(dialog)
	}
	path, err := onePath(dialog)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func prepare(dialog uintptr, req filedialog.Request) (dialogText, error) {
	var kept dialogText
	var opts uint32
	if hr, err := comCall(dialog, slotGetOptions, uintptr(unsafe.Pointer(&opts))); err != nil {
		return kept, statusErr("options", hr, err)
	}
	opts |= fosForceFileSystem | fosNoChangeDir
	if req.Folder {
		opts |= fosPickFolders
	}
	if req.Multiple {
		opts |= fosAllowMulti
	}
	if req.Save {
		opts |= fosOverwrite
	} else if !req.Folder {
		opts |= fosFileMustExist | fosPathMustExist
	}
	if hr, err := comCall(dialog, slotSetOptions, uintptr(opts)); err != nil {
		return kept, statusErr("options", hr, err)
	}
	title, err := syscall.UTF16PtrFromString(req.TitleOrDefault())
	if err != nil {
		return kept, err
	}
	kept.title = title
	if hr, err := comCall(dialog, slotSetTitle, uintptr(unsafe.Pointer(title))); err != nil {
		return kept, statusErr("title", hr, err)
	}
	if req.Save && req.Name != "" {
		name, err := syscall.UTF16PtrFromString(req.Name)
		if err != nil {
			return kept, err
		}
		kept.name = name
		if hr, err := comCall(dialog, slotSetFileName, uintptr(unsafe.Pointer(name))); err != nil {
			return kept, statusErr("name", hr, err)
		}
	}
	if req.Directory != "" {
		if err := setFolder(dialog, req.Directory); err != nil {
			return kept, err
		}
	}
	if err := setFilters(dialog, req, &kept); err != nil {
		return kept, err
	}
	return kept, nil
}

func setFolder(dialog uintptr, directory string) error {
	path, err := syscall.UTF16PtrFromString(directory)
	if err != nil {
		return err
	}
	var item uintptr
	hr, _, callErr := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(path)),
		0,
		uintptr(unsafe.Pointer(&iidItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	runtime.KeepAlive(path)
	if failed(hr) || item == 0 {
		return statusErr("folder", hr, callErr)
	}
	defer func() {
		if releaseErr := release(item); releaseErr != nil {
			return
		}
	}()
	if hr, err = comCall(dialog, slotSetFolder, item); err != nil {
		return statusErr("folder", hr, err)
	}
	return nil
}

func setFilters(dialog uintptr, req filedialog.Request, kept *dialogText) error {
	for _, item := range req.Filters {
		var globs []string
		for _, pattern := range item.Patterns {
			if pattern != "" {
				globs = append(globs, pattern)
			}
		}
		if len(globs) == 0 {
			continue
		}
		name := item.Name
		if name == "" {
			name = globs[0]
		}
		nameUTF, err := syscall.UTF16FromString(name)
		if err != nil {
			return err
		}
		specUTF, err := syscall.UTF16FromString(strings.Join(globs, ";"))
		if err != nil {
			return err
		}
		kept.text = append(kept.text, nameUTF, specUTF)
		n := len(kept.text)
		kept.specs = append(kept.specs, filterSpec{name: &kept.text[n-2][0], spec: &kept.text[n-1][0]})
	}
	if len(kept.specs) == 0 {
		return nil
	}
	if hr, err := comCall(dialog, slotSetFileTypes, uintptr(len(kept.specs)), uintptr(unsafe.Pointer(&kept.specs[0]))); err != nil {
		return statusErr("filter", hr, err)
	}
	return nil
}

func onePath(dialog uintptr) (string, error) {
	var item uintptr
	if hr, err := comCall(dialog, slotGetResult, uintptr(unsafe.Pointer(&item))); err != nil || item == 0 {
		return "", statusErr("result", hr, err)
	}
	defer func() {
		if releaseErr := release(item); releaseErr != nil {
			return
		}
	}()
	return itemPath(item)
}

func collectMany(dialog uintptr) ([]string, error) {
	var items uintptr
	if hr, err := comCall(dialog, slotGetResults, uintptr(unsafe.Pointer(&items))); err != nil || items == 0 {
		return nil, statusErr("result", hr, err)
	}
	defer func() {
		if releaseErr := release(items); releaseErr != nil {
			return
		}
	}()
	var count uint32
	if hr, err := comCall(items, 7, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil, statusErr("result", hr, err)
	}
	paths := make([]string, 0, count)
	for i := range count {
		var item uintptr
		if hr, err := comCall(items, 8, uintptr(i), uintptr(unsafe.Pointer(&item))); err != nil || item == 0 {
			return nil, statusErr("result", hr, err)
		}
		path, err := itemPath(item)
		release(item)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: result", errDialog)
	}
	return paths, nil
}

func itemPath(item uintptr) (string, error) {
	var name *uint16
	hr, err := comCall(item, 5, sigdnFileSysPath, uintptr(unsafe.Pointer(&name)))
	if err != nil || name == nil {
		return "", statusErr("path", hr, err)
	}
	defer func() {
		_, _, freeErr := procCoTaskMemFree.Call(uintptr(unsafe.Pointer(name)))
		if freeErr != nil {
			return
		}
	}()
	return utf16z(name), nil
}

func release(obj uintptr) error {
	if obj == 0 {
		return nil
	}
	_, err := comCall(obj, 2)
	return err
}

func comCall(obj uintptr, slot uintptr, args ...uintptr) (uintptr, error) {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + slot*unsafe.Sizeof(uintptr(0))))
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, obj)
	all = append(all, args...)
	hr, _, callErr := syscall.SyscallN(fn, all...)
	if failed(hr) {
		return hr, statusErr("com", hr, callErr)
	}
	return hr, nil
}

func statusErr(op string, hr uintptr, callErr error) error {
	if callErr == nil {
		return fmt.Errorf("%w: %s 0x%x", errDialog, op, uint32(hr))
	}
	return fmt.Errorf("%w: %s 0x%x: %w", errDialog, op, uint32(hr), callErr)
}

func failed(hr uintptr) bool {
	return int32(hr) < 0
}

func ownerWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

func presentDialog(dialog uintptr) (uintptr, error, uintptr) {
	// NULL owner. The WebView2 HWND belongs to another thread, and passing
	// it makes Show return E_FAIL.
	hr, err := showDialog(dialog, 0)
	if err == nil || uint32(hr) == hrCanceled || uint32(hr) != eFail {
		return hr, err, 0
	}
	local := createOwner()
	if local == 0 {
		return hr, err, 0
	}
	hr, err = showDialog(dialog, local)
	return hr, err, local
}

func showDialog(dialog, owner uintptr) (uintptr, error) {
	if owner != 0 {
		procAllowSetForegroundWindow.Call(uintptr(0xFFFFFFFF))
		detach := attachForeground(owner)
		defer detach()
	}
	return comCall(dialog, slotShow, owner)
}

func attachForeground(target uintptr) func() {
	fg, _, _ := procGetForegroundWindow.Call()
	var fgThread uintptr
	if fg != 0 {
		fgThread, _, _ = procGetWindowThreadProcessId.Call(fg, 0)
	}
	our, _, _ := procGetCurrentThreadId.Call()
	attached := false
	if fgThread != 0 && fgThread != our {
		r, _, _ := procAttachThreadInput.Call(our, fgThread, 1)
		attached = r != 0
	}
	if target != 0 {
		procSetForegroundWindow.Call(target)
		procBringWindowToTop.Call(target)
	}
	return func() {
		if attached {
			procAttachThreadInput.Call(our, fgThread, 0)
		}
	}
}

func createOwner() uintptr {
	ownerOnce.Do(func() {
		inst, _, _ := procGetModuleHandleW.Call(0)
		class := ownerClassEx{
			wndProc:   ownerProc,
			instance:  inst,
			className: ownerName,
		}
		class.size = uint32(unsafe.Sizeof(class))
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
	})
	inst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(ownerName)),
		uintptr(unsafe.Pointer(ownerTitle)),
		wsOverlappedWindow|wsVisible,
		uintptr(cwUseDefault), uintptr(cwUseDefault), 480, 240,
		0, 0, inst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	procShowWindow.Call(hwnd, swShowNormal)
	procSetForegroundWindow.Call(hwnd)
	return hwnd
}

func utf16z(p *uint16) string {
	if p == nil {
		return ""
	}
	var buf []uint16
	for {
		c := *p
		if c == 0 {
			break
		}
		buf = append(buf, c)
		p = (*uint16)(unsafe.Add(unsafe.Pointer(p), 2))
	}
	return syscall.UTF16ToString(buf)
}
