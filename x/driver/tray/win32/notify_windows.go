//go:build windows

package win32

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/tray"
)

const (
	wmDestroy      = 0x0002
	wmClose        = 0x0010
	wmNull         = 0x0000
	wmLButtonUp    = 0x0202
	wmRButtonUp    = 0x0205
	wmContextMenu  = 0x007B
	wmDo           = 0x8000 + 20
	trayMessage    = 0x8000 + 21
	nimAdd         = 0
	nimModify      = 1
	nimDelete      = 2
	nifMessage     = 0x1
	nifIcon        = 0x2
	nifTip         = 0x4
	mfString       = 0
	mfDisabled     = 0x2
	mfChecked      = 0x8
	mfPopup        = 0x10
	mfSeparator    = 0x800
	tpmReturnCmd   = 0x100
	tpmNonotify    = 0x80
	tpmRightButton = 0x2
	dibRGBColors   = 0
	hwndMessage    = ^uintptr(2)
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procAppendMenuW         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	procCreateBitmap        = gdi32.NewProc("CreateBitmap")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procCreateIconIndirect  = user32.NewProc("CreateIconIndirect")
	procDestroyIcon         = user32.NewProc("DestroyIcon")
	procShellNotifyIconW    = shell32.NewProc("Shell_NotifyIconW")

	classOnce sync.Once
	classAtom uintptr
	classErr  error
	className = syscall.StringToUTF16Ptr("lewkit.driver.tray")
	windows   sync.Map
)

var (
	errModule  = errors.New("tray module")
	errWindow  = errors.New("tray window")
	errClass   = errors.New("tray window class")
	errDestroy = errors.New("destroy tray window")
	errNotify  = errors.New("shell notify icon")
	errPopup   = errors.New("popup menu")
	errAppend  = errors.New("append menu")
	errBitmap  = errors.New("tray bitmap")
	errMask    = errors.New("tray mask")
	errIcon    = errors.New("tray icon")
)

type opener struct{}

func (opener) Open(ctx context.Context, cfg tray.Config) (tray.Tray, error) {
	item := &statusItem{cfg: cfg}
	item.ready = make(chan error)
	go item.loop()
	if err := <-item.ready; err != nil {
		return nil, err
	}
	context.AfterFunc(ctx, func() {
		if err := item.Close(); err != nil {
			return
		}
	})
	return item, nil
}

type job struct {
	fn   func() error
	done chan error
}

type statusItem struct {
	mu       sync.Mutex
	cfg      tray.Config
	hwnd     uintptr
	icon     uintptr
	queue    []job
	closed   bool
	ready    chan error
	closeOne sync.Once
}

type wndClassEx struct {
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

type point struct{ x, y int32 }

type message struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type notifyIconData struct {
	size            uint32
	window          uintptr
	id              uint32
	flags           uint32
	callbackMessage uint32
	icon            uintptr
	tip             [128]uint16
}

type bitmapInfoHeader struct {
	size          uint32
	width         int32
	height        int32
	planes        uint16
	bitCount      uint16
	compression   uint32
	sizeImage     uint32
	xPelsPerMeter int32
	yPelsPerMeter int32
	clrUsed       uint32
	clrImportant  uint32
}

type iconInfo struct {
	icon     int32
	xHotspot int32
	yHotspot int32
	mask     uintptr
	color    uintptr
}

func (item *statusItem) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := item.create(); err != nil {
		item.ready <- err
		return
	}
	item.ready <- nil
	var msg message
	for {
		code, _, callErr := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(code) < 0 {
			if callErr != nil {
				return
			}
			return
		}
		if code == 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (item *statusItem) create() error {
	classOnce.Do(registerClass)
	if classErr != nil {
		return classErr
	}
	inst, _, moduleErr := procGetModuleHandleW.Call(0)
	if inst == 0 {
		if moduleErr == nil || moduleErr == syscall.Errno(0) {
			return errModule
		}
		return moduleErr
	}
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, hwndMessage, 0, inst, 0)
	if hwnd == 0 {
		return fmt.Errorf("%w: %w", errWindow, err)
	}
	item.hwnd = hwnd
	windows.Store(hwnd, item)
	if err := item.apply(item.cfg); err != nil {
		windows.Delete(hwnd)
		if destroyed, _, destroyErr := procDestroyWindow.Call(hwnd); destroyed == 0 && destroyErr != nil && destroyErr != syscall.Errno(0) {
			return errors.Join(err, destroyErr)
		}
		return err
	}
	return nil
}

func registerClass() {
	inst, _, moduleErr := procGetModuleHandleW.Call(0)
	if inst == 0 {
		if moduleErr == nil || moduleErr == syscall.Errno(0) {
			classErr = errModule
			return
		}
		classErr = moduleErr
		return
	}
	class := wndClassEx{
		size:      uint32(unsafe.Sizeof(wndClassEx{})),
		wndProc:   syscall.NewCallback(wndProc),
		instance:  inst,
		className: className,
	}
	classAtom, _, classErr = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
	if classAtom == 0 {
		classErr = fmt.Errorf("%w: %w", errClass, classErr)
		return
	}
	classErr = nil
}

func (item *statusItem) Update(cfg tray.Config) error {
	if cfg.ID == "" {
		item.mu.Lock()
		cfg.ID = item.cfg.ID
		item.mu.Unlock()
	}
	return item.onWindow(func() error {
		return item.apply(cfg)
	})
}

func (item *statusItem) Close() error {
	var err error
	item.closeOne.Do(func() {
		err = item.onWindow(func() error {
			item.remove()
			if destroyed, _, callErr := procDestroyWindow.Call(item.hwnd); destroyed == 0 {
				if callErr == nil || callErr == syscall.Errno(0) {
					return errDestroy
				}
				return callErr
			}
			if posted, _, callErr := procPostQuitMessage.Call(0); posted == 0 && callErr != nil && callErr != syscall.Errno(0) {
				return callErr
			}
			return nil
		})
	})
	return err
}

func (item *statusItem) onWindow(fn func() error) error {
	done := make(chan error, 1)
	item.mu.Lock()
	if item.closed || item.hwnd == 0 {
		item.mu.Unlock()
		return tray.ErrClosed
	}
	item.queue = append(item.queue, job{fn: fn, done: done})
	hwnd := item.hwnd
	item.mu.Unlock()
	procPostMessageW.Call(hwnd, wmDo, 0, 0)
	return <-done
}

func (item *statusItem) drain() {
	for {
		item.mu.Lock()
		if len(item.queue) == 0 {
			item.mu.Unlock()
			return
		}
		job := item.queue[0]
		item.queue = item.queue[1:]
		item.mu.Unlock()
		job.done <- job.fn()
	}
}

func (item *statusItem) apply(cfg tray.Config) error {
	icon, err := iconHandle(cfg.Icon)
	if err != nil {
		return err
	}
	data := notifyIconData{
		size:            uint32(unsafe.Sizeof(notifyIconData{})),
		window:          item.hwnd,
		id:              1,
		flags:           nifMessage | nifIcon | nifTip,
		callbackMessage: trayMessage,
		icon:            icon,
	}
	copyTip(&data, tray.Tip(cfg))
	action := uintptr(nimModify)
	if item.icon == 0 {
		action = nimAdd
	}
	ok, _, callErr := procShellNotifyIconW.Call(action, uintptr(unsafe.Pointer(&data)))
	if ok == 0 {
		procDestroyIcon.Call(icon)
		if callErr == nil || callErr == syscall.Errno(0) {
			return errNotify
		}
		return fmt.Errorf("%w: %w", errNotify, callErr)
	}
	if item.icon != 0 {
		procDestroyIcon.Call(item.icon)
	}
	item.icon = icon
	item.mu.Lock()
	item.cfg = cfg
	item.mu.Unlock()
	return nil
}

func (item *statusItem) remove() {
	if item.hwnd == 0 {
		return
	}
	data := notifyIconData{
		size:   uint32(unsafe.Sizeof(notifyIconData{})),
		window: item.hwnd,
		id:     1,
	}
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	if item.icon != 0 {
		procDestroyIcon.Call(item.icon)
		item.icon = 0
	}
	windows.Delete(item.hwnd)
	item.mu.Lock()
	item.closed = true
	item.mu.Unlock()
}

func (item *statusItem) showMenu() {
	item.mu.Lock()
	nodes := tray.Tree(item.cfg.Menu)
	item.mu.Unlock()
	if len(nodes) == 0 {
		return
	}
	menu, err := buildMenu(nodes)
	if err != nil || menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	var cursor point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	procSetForegroundWindow.Call(item.hwnd)
	id, _, callErr := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmNonotify|tpmRightButton, uintptr(cursor.x), uintptr(cursor.y), 0, item.hwnd, 0)
	if id == 0 && callErr != nil && callErr != syscall.Errno(0) {
		return
	}
	procPostMessageW.Call(item.hwnd, wmNull, 0, 0)
	if id == 0 {
		return
	}
	clicked, ok := tray.Find(nodes, int(id))
	if !ok || clicked.OnClick == nil || clicked.Separator || len(clicked.Children) > 0 {
		return
	}
	go clicked.OnClick()
}

func buildMenu(nodes []tray.Node) (uintptr, error) {
	menu, _, err := procCreatePopupMenu.Call()
	if menu == 0 {
		return 0, fmt.Errorf("%w: %w", errPopup, err)
	}
	for _, node := range nodes {
		if err := appendItem(menu, node); err != nil {
			procDestroyMenu.Call(menu)
			return 0, err
		}
	}
	return menu, nil
}

func appendItem(menu uintptr, node tray.Node) error {
	item := node.Item
	if item.Separator {
		procAppendMenuW.Call(menu, mfSeparator, 0, 0)
		return nil
	}
	title, err := syscall.UTF16PtrFromString(item.Label)
	if err != nil {
		return err
	}
	flags := uintptr(mfString)
	id := uintptr(node.ID)
	if len(node.Children) > 0 {
		child, err := buildMenu(node.Children)
		if err != nil {
			return err
		}
		flags = mfPopup
		id = child
	}
	if item.Disabled {
		flags |= mfDisabled
	}
	if item.Checked {
		flags |= mfChecked
	}
	ok, _, callErr := procAppendMenuW.Call(menu, flags, id, uintptr(unsafe.Pointer(title)))
	if ok == 0 {
		return fmt.Errorf("%w: %w", errAppend, callErr)
	}
	return nil
}

func copyTip(data *notifyIconData, tip string) {
	encoded, err := syscall.UTF16FromString(tip)
	if err != nil {
		return
	}
	if len(encoded) > len(data.tip) {
		encoded = encoded[:len(data.tip)-1]
		encoded = append(encoded, 0)
	}
	copy(data.tip[:], encoded)
}

func iconHandle(icon tray.Icon) (uintptr, error) {
	img, err := tray.Raster(icon, 32)
	if err != nil {
		return 0, err
	}
	if img == nil {
		img = image.NewNRGBA(image.Rect(0, 0, 16, 16))
		for y := range 16 {
			for x := range 16 {
				img.SetNRGBA(x, y, color.NRGBA{R: 40, G: 40, B: 40, A: 255})
			}
		}
	}
	return iconFromNRGBA(img)
}

func iconFromNRGBA(img *image.NRGBA) (uintptr, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	header := bitmapInfoHeader{
		size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		width:    int32(width),
		height:   -int32(height),
		planes:   1,
		bitCount: 32,
	}
	var bits unsafe.Pointer
	colorBitmap, _, err := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&header)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if colorBitmap == 0 || bits == nil {
		return 0, fmt.Errorf("%w: %w", errBitmap, err)
	}
	pixels := unsafe.Slice((*byte)(bits), width*height*4)
	for y := range height {
		for x := range width {
			pixel := img.NRGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			alpha := uint32(pixel.A)
			i := (y*width + x) * 4
			pixels[i] = byte(uint32(pixel.B) * alpha / 255)
			pixels[i+1] = byte(uint32(pixel.G) * alpha / 255)
			pixels[i+2] = byte(uint32(pixel.R) * alpha / 255)
			pixels[i+3] = pixel.A
		}
	}
	stride := ((width + 31) / 32) * 4
	maskBits := make([]byte, stride*height)
	mask, _, maskErr := procCreateBitmap.Call(uintptr(width), uintptr(height), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	if mask == 0 {
		procDeleteObject.Call(colorBitmap)
		return 0, fmt.Errorf("%w: %w", errMask, maskErr)
	}
	info := iconInfo{icon: 1, mask: mask, color: colorBitmap}
	handle, _, iconErr := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	procDeleteObject.Call(colorBitmap)
	procDeleteObject.Call(mask)
	if handle == 0 {
		return 0, fmt.Errorf("%w: %w", errIcon, iconErr)
	}
	return handle, nil
}

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	value, ok := windows.Load(hwnd)
	if !ok {
		ret, _, callErr := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		if ret == 0 && callErr == syscall.Errno(0) {
			return 0
		}
		return ret
	}
	item := value.(*statusItem)
	switch msg {
	case wmDo:
		item.drain()
		return 0
	case trayMessage:
		switch lparam {
		case wmLButtonUp, wmRButtonUp, wmContextMenu:
			item.showMenu()
		}
		return 0
	case wmClose, wmDestroy:
		item.remove()
		return 0
	default:
		ret, _, callErr := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		if ret == 0 && callErr == syscall.Errno(0) {
			return 0
		}
		return ret
	}
}
