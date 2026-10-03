//go:build windows

package win32

import (
	"context"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/ffi/native"

	_ "github.com/lewtec/lewkit/x/driver/thread/std"
)

const (
	wmInitDialog = 0x0110
	wmCommand    = 0x0111
	wmGetText    = 0x000D
	lbAddString  = 0x0180
	lbGetCurSel  = 0x0188
	lbGetText    = 0x0189
	lbSetCurSel  = 0x0186
	mbYesNo      = 0x0004
	mbIconQuest  = 0x0020
	idYes        = 6
)

var (
	procDialogBox           = native.ProcOf("user32.dll", "DialogBoxIndirectParamW")
	procEndDialog           = native.ProcOf("user32.dll", "EndDialog")
	procGetDlgItem          = native.ProcOf("user32.dll", "GetDlgItem")
	procSendMessageW        = native.ProcOf("user32.dll", "SendMessageW")
	procMessageBox          = native.ProcOf("user32.dll", "MessageBoxW")
	procGetModuleHandleW    = native.ProcOf("kernel32.dll", "GetModuleHandleW")
	dialogProcedureCallback = syscall.NewCallback(dialogProcedure)
)

type dlgState struct {
	labels []string
	edit   bool
	text   string
}

var state dlgState

func ask(ctx context.Context, kind, title, body string, items []string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if thread.Bound() && !thread.On() {
		var text string
		var ok bool
		var err error
		thread.Do(func() {
			text, ok, err = askHere(kind, title, body, items)
		})
		return text, ok, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return askHere(kind, title, body, items)
}

func askHere(kind, title, body string, items []string) (string, bool, error) {
	if kind == "confirm" {
		return "", confirmBox(title), nil
	}
	state = dlgState{labels: items, edit: kind == "prompt", text: body}
	var tpl []byte
	if kind == "choose" {
		tpl = chooseDialog(title).bytes()
	} else {
		tpl = promptDialog(title, body).bytes()
	}
	inst, _, _ := procGetModuleHandleW.Call(0)
	r, _, callErr := procDialogBox.Call(inst, uintptr(unsafe.Pointer(&tpl[0])), 0, dialogProcedureCallback, 0)
	text := state.text
	state = dlgState{}
	runtime.KeepAlive(tpl)
	if int32(r) == -1 {
		return "", false, callErr
	}
	if int32(r) != idOK {
		return "", false, nil
	}
	return text, true, nil
}

func confirmBox(message string) bool {
	if message == "" {
		message = "Confirm"
	}
	text, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return false
	}
	caption, err := syscall.UTF16PtrFromString("Confirm")
	if err != nil {
		return false
	}
	r, _, _ := procMessageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(caption)), mbYesNo|mbIconQuest)
	return r == idYes
}

func dialogProcedure(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case wmInitDialog:
		if state.edit {
			setText(item(hwnd, idField), state.text)
			return 1
		}
		list := item(hwnd, idField)
		for _, label := range state.labels {
			ptr, err := syscall.UTF16PtrFromString(label)
			if err != nil {
				continue
			}
			procSendMessageW.Call(list, lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
			runtime.KeepAlive(ptr)
		}
		procSendMessageW.Call(list, lbSetCurSel, 0, 0)
		return 1
	case wmCommand:
		switch wparam & 0xFFFF {
		case idOK:
			if state.edit {
				state.text = getText(item(hwnd, idField))
			} else {
				state.text = selected(item(hwnd, idField))
				if state.text == "" {
					procEndDialog.Call(hwnd, idCancel)
					return 1
				}
			}
			procEndDialog.Call(hwnd, idOK)
			return 1
		case idCancel:
			procEndDialog.Call(hwnd, idCancel)
			return 1
		}
	}
	_ = lparam
	return 0
}

func item(hwnd, id uintptr) uintptr {
	h, _, _ := procGetDlgItem.Call(hwnd, id)
	return h
}

func setText(hwnd uintptr, text string) {
	ptr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	procSendMessageW.Call(hwnd, 0x000C, 0, uintptr(unsafe.Pointer(ptr)))
	runtime.KeepAlive(ptr)
}

func getText(hwnd uintptr) string {
	buf := make([]uint16, 1024)
	procSendMessageW.Call(hwnd, wmGetText, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

func selected(list uintptr) string {
	sel, _, _ := procSendMessageW.Call(list, lbGetCurSel, 0, 0)
	if int32(sel) < 0 {
		return ""
	}
	buf := make([]uint16, 512)
	procSendMessageW.Call(list, lbGetText, sel, uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}
