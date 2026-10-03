//go:build windows

package windows

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	nimAdd         = 0
	nimModify      = 1
	nimDelete      = 2
	nifIcon        = 0x2
	nifTip         = 0x4
	nifInfo        = 0x10
	idiInformation = 32516
	hwndMessage    = ^uintptr(2)
)

var (
	procShellNotifyIconW = native.ProcOf("shell32.dll", "Shell_NotifyIconW")
	procLoadIconW        = native.ProcOf("user32.dll", "LoadIconW")
	procRegisterClassExW = native.ProcOf("user32.dll", "RegisterClassExW")
	procCreateWindowExW  = native.ProcOf("user32.dll", "CreateWindowExW")
	procDefWindowProcW   = native.ProcOf("user32.dll", "DefWindowProcW")
	procGetModuleHandleW = native.ProcOf("kernel32.dll", "GetModuleHandleW")

	notifyOnce   sync.Once
	notifyWindow uintptr
	notifyIcon   uintptr
	notifyClass  = syscall.StringToUTF16Ptr("lewkit.notification")
	notifyProc   = syscall.NewCallback(notifyWndProc)

	notifyMu sync.Mutex
	added    = map[uint32]uint64{}
	nextGen  uint64
)

type notifyClassEx struct {
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

func available(context.Context) error { return driver.ForGOOS("windows") }

func (backend) Notify(ctx context.Context, n notification.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	hwnd, icon, err := ensureNotifyWindow()
	if err != nil {
		return err
	}
	id := n.ID
	if id == 0 {
		id = 1
	}
	title := n.Title
	if title == "" {
		title = "Notification"
	}
	var data notifyIconData
	data.size = uint32(unsafe.Sizeof(data))
	data.window = hwnd
	data.id = id
	data.flags = nifIcon | nifTip | nifInfo
	data.icon = icon
	copyUTF16(data.tip[:], title)
	copyUTF16(data.info[:], n.Message)
	copyUTF16(data.infoTitle[:], title)
	data.infoFlags = infoFlags(n.Urgency)

	notifyMu.Lock()
	defer notifyMu.Unlock()
	if err := shellUpdate(&data, id); err != nil {
		return err
	}
	nextGen++
	gen := nextGen
	added[id] = gen
	time.AfterFunc(8*time.Second, func() { retire(id, gen) })
	return nil
}

func shellUpdate(data *notifyIconData, id uint32) error {
	cmd := uintptr(nimAdd)
	if _, ok := added[id]; ok {
		cmd = nimModify
	}
	if shell(cmd, data) == nil {
		return nil
	}
	other := uintptr(nimModify)
	if cmd == nimModify {
		other = nimAdd
	}
	if err := shell(other, data); err != nil {
		return err
	}
	return nil
}

func shell(cmd uintptr, data *notifyIconData) error {
	r, _, callErr := procShellNotifyIconW.Call(cmd, uintptr(unsafe.Pointer(data)))
	if r == 0 {
		if callErr == nil {
			return fmt.Errorf("notify icon")
		}
		return fmt.Errorf("notify icon: %w", callErr)
	}
	return nil
}

func retire(id uint32, gen uint64) {
	notifyMu.Lock()
	defer notifyMu.Unlock()
	if added[id] != gen {
		return
	}
	var data notifyIconData
	data.size = uint32(unsafe.Sizeof(data))
	data.window = notifyWindow
	data.id = id
	_ = shell(nimDelete, &data)
	delete(added, id)
}

func ensureNotifyWindow() (uintptr, uintptr, error) {
	var err error
	notifyOnce.Do(func() {
		inst, _, _ := procGetModuleHandleW.Call(0)
		class := notifyClassEx{
			wndProc:   notifyProc,
			instance:  inst,
			className: notifyClass,
		}
		class.size = uint32(unsafe.Sizeof(class))
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
		hwnd, _, callErr := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(notifyClass)),
			0,
			0,
			0, 0, 0, 0,
			hwndMessage, 0, inst, 0,
		)
		if hwnd == 0 {
			err = fmt.Errorf("notify window: %v", callErr)
			return
		}
		icon, _, _ := procLoadIconW.Call(0, idiInformation)
		if icon == 0 {
			err = fmt.Errorf("notify icon")
			return
		}
		notifyWindow = hwnd
		notifyIcon = icon
	})
	if notifyWindow == 0 {
		if err == nil {
			err = fmt.Errorf("%w: notify window", driver.ErrUnavailable)
		}
		return 0, 0, err
	}
	return notifyWindow, notifyIcon, nil
}

func notifyWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

func copyUTF16(dst []uint16, text string) {
	if len(dst) == 0 {
		return
	}
	encoded, err := syscall.UTF16FromString(text)
	if err != nil || len(encoded) == 0 {
		dst[0] = 0
		return
	}
	n := copy(dst, encoded)
	dst[n-1] = 0
	if n < len(encoded) {
		dst[len(dst)-1] = 0
	}
}
