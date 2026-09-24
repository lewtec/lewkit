//go:build windows

package win32

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/thread"
)

const (
	coinitApartment    = 2
	clsctxInprocServer = 1
	sigdnFileSysPath   = 0x80058000
	fosOverwrite       = 0x2
	fosNoChangeDir     = 0x8
	fosPickFolders     = 0x20
	fosForceFileSystem = 0x40
	fosAllowMulti      = 0x200
	fosPathMustExist   = 0x800
	fosFileMustExist   = 0x1000
	hrCanceled         = 0x800704C7
)

var errDialog = errors.New("file dialog")

var (
	clsidOpen = syscall.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	clsidSave = syscall.GUID{Data1: 0xC0B4E2F3, Data2: 0xBA21, Data3: 0x4773, Data4: [8]byte{0x8D, 0xBA, 0x33, 0x5E, 0xC9, 0x46, 0xEB, 0x8B}}
	iidOpen   = syscall.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidSave   = syscall.GUID{Data1: 0x84BCCD23, Data2: 0x5FDE, Data3: 0x4CDB, Data4: [8]byte{0xAE, 0xA4, 0xAF, 0x64, 0xB8, 0x3D, 0x78, 0xAB}}
	iidItem   = syscall.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}

	ole32                           = syscall.NewLazyDLL("ole32.dll")
	shell32                         = syscall.NewLazyDLL("shell32.dll")
	procCoInitializeEx              = ole32.NewProc("CoInitializeEx")
	procCoUninitialize              = ole32.NewProc("CoUninitialize")
	procCoCreateInstance            = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree               = ole32.NewProc("CoTaskMemFree")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
)

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
	if !thread.Bound() {
		return nil, fmt.Errorf("%w: thread not bound", errDialog)
	}
	var (
		paths []string
		err   error
	)
	thread.Do(func() {
		paths, err = show(req)
	})
	return paths, err
}

func show(req filedialog.Request) ([]string, error) {
	hr, _, callErr := procCoInitializeEx.Call(0, coinitApartment)
	if int32(hr) < 0 {
		return nil, statusErr("com", hr, callErr)
	}
	if hr == 0 {
		defer func() {
			_, _, uninitErr := procCoUninitialize.Call()
			if uninitErr != nil {
				return
			}
		}()
	}

	class, iid := &clsidOpen, &iidOpen
	if req.Save {
		class, iid = &clsidSave, &iidSave
	}
	var dialog uintptr
	hr, _, callErr = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(class)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if failed(hr) || dialog == 0 {
		return nil, statusErr("dialog", hr, callErr)
	}
	defer func() {
		if releaseErr := release(dialog); releaseErr != nil {
			return
		}
	}()

	if err := prepare(dialog, req); err != nil {
		return nil, err
	}
	hr, err := comCall(dialog, 3, 0)
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

func prepare(dialog uintptr, req filedialog.Request) error {
	var opts uint32
	if hr, err := comCall(dialog, 10, uintptr(unsafe.Pointer(&opts))); err != nil {
		return statusErr("options", hr, err)
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
	if hr, err := comCall(dialog, 9, uintptr(opts)); err != nil {
		return statusErr("options", hr, err)
	}
	title, err := syscall.UTF16PtrFromString(req.TitleOrDefault())
	if err != nil {
		return err
	}
	if hr, err := comCall(dialog, 16, uintptr(unsafe.Pointer(title))); err != nil {
		return statusErr("title", hr, err)
	}
	if req.Save && req.Name != "" {
		name, err := syscall.UTF16PtrFromString(req.Name)
		if err != nil {
			return err
		}
		if hr, err := comCall(dialog, 14, uintptr(unsafe.Pointer(name))); err != nil {
			return statusErr("name", hr, err)
		}
	}
	if req.Directory != "" {
		if err := setFolder(dialog, req.Directory); err != nil {
			return err
		}
	}
	return setFilters(dialog, req)
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
	if failed(hr) || item == 0 {
		return statusErr("folder", hr, callErr)
	}
	defer func() {
		if releaseErr := release(item); releaseErr != nil {
			return
		}
	}()
	if hr, err = comCall(dialog, 12, item); err != nil {
		return statusErr("folder", hr, err)
	}
	return nil
}

func setFilters(dialog uintptr, req filedialog.Request) error {
	var specs []filterSpec
	var kept [][]uint16
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
		kept = append(kept, nameUTF, specUTF)
		specs = append(specs, filterSpec{name: &nameUTF[0], spec: &specUTF[0]})
	}
	if len(specs) == 0 {
		return nil
	}
	if hr, err := comCall(dialog, 4, uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0]))); err != nil {
		return statusErr("filter", hr, err)
	}
	runtime.KeepAlive(kept)
	runtime.KeepAlive(specs)
	return nil
}

func onePath(dialog uintptr) (string, error) {
	var item uintptr
	if hr, err := comCall(dialog, 19, uintptr(unsafe.Pointer(&item))); err != nil || item == 0 {
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
	if hr, err := comCall(dialog, 26, uintptr(unsafe.Pointer(&items))); err != nil || items == 0 {
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
