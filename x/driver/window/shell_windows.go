//go:build windows

package window

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/image/convert"
)

const (
	wmSetIcon = 0x0080
	iconSmall = 0
	iconBig   = 1
	dibRGB    = 0
)

var (
	procCreateDIBSection   = native.ProcOf("gdi32.dll", "CreateDIBSection")
	procCreateBitmap       = native.ProcOf("gdi32.dll", "CreateBitmap")
	procCreateIconIndirect = native.ProcOf("user32.dll", "CreateIconIndirect")
	procDeleteObject       = native.ProcOf("gdi32.dll", "DeleteObject")
	procSendMessageW       = native.ProcOf("user32.dll", "SendMessageW")

	iconOnce   sync.Once
	iconBigH   uintptr
	iconSmallH uintptr
)

// ApplyWindowIcon sets the taskbar and Alt-Tab icons for hwnd.
// The handles live until the process exits.
func ApplyWindowIcon(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	big, small := windowIcons()
	if big != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconBig, big)
	}
	if small != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, small)
	}
}

func showShell(string, image.Image) {}

func windowIcons() (uintptr, uintptr) {
	iconOnce.Do(loadWindowIcons)
	return iconBigH, iconSmallH
}

func loadWindowIcons() {
	src := shellImage(nil)
	if src == nil {
		return
	}
	big, err := convert.Square(src, 256)
	if err != nil {
		return
	}
	small, err := convert.Square(src, 32)
	if err != nil {
		return
	}
	iconBigH, err = hicon(big)
	if err != nil {
		iconBigH = 0
	}
	iconSmallH, err = hicon(small)
	if err != nil {
		iconSmallH = 0
	}
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

var errGDI = errors.New("gdi")

func hicon(img *image.NRGBA) (uintptr, error) {
	if img == nil {
		return 0, errGDI
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 1 || height < 1 {
		return 0, errGDI
	}
	header := bitmapInfoHeader{
		size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		width:    int32(width),
		height:   -int32(height),
		planes:   1,
		bitCount: 32,
	}
	var bits unsafe.Pointer
	colorBitmap, extra, callErr := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&header)), dibRGB, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if colorBitmap == 0 || bits == nil {
		if colorBitmap != 0 {
			procDeleteObject.Call(colorBitmap)
		}
		return 0, gdiErr("create dib section", extra, callErr)
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
	mask, maskExtra, maskErr := procCreateBitmap.Call(uintptr(width), uintptr(height), 1, 1, uintptr(unsafe.Pointer(&maskBits[0])))
	if mask == 0 {
		procDeleteObject.Call(colorBitmap)
		return 0, gdiErr("create bitmap", maskExtra, maskErr)
	}
	info := iconInfo{icon: 1, mask: mask, color: colorBitmap}
	handle, iconExtra, iconErr := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	procDeleteObject.Call(colorBitmap)
	procDeleteObject.Call(mask)
	if handle == 0 {
		return 0, gdiErr("create icon", iconExtra, iconErr)
	}
	return handle, nil
}

func gdiErr(op string, extra uintptr, err error) error {
	if err == nil || err == syscall.Errno(0) {
		err = errGDI
	}
	if extra != 0 {
		return fmt.Errorf("%s status %d: %w", op, extra, err)
	}
	return fmt.Errorf("%s: %w", op, err)
}
