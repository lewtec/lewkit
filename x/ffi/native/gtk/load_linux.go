//go:build linux

package gtk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means GTK 4 is missing or this process cannot open a display.
var ErrUnavailable = errors.New("gtk unavailable")

// errMixedGTK means this process is on GTK 3, so GTK 4 must not be loaded.
var errMixedGTK = errors.New("GTK 4 cannot share a process with GTK 3")

const (
	orientVertical   = 1
	policyNever      = 2
	policyAutomatic  = 1
	listBoxAppendPos = -1
)

type symbols struct {
	initCheck    func() int32
	setPrgname   func(*byte)
	iterate      func(ctx uintptr, block int32) int32
	signal       func(obj uintptr, name *byte, handler, data, destroy uintptr, flags uint32) uint64
	windowNew    func() uintptr
	setTitle     func(win uintptr, title *byte)
	setSize      func(win uintptr, w, h int32)
	setModal     func(win uintptr, modal int32)
	setChild     func(win, child uintptr)
	setResizable func(win uintptr, resizable int32)
	setDefault   func(win, widget uintptr)
	present      func(win uintptr)
	destroy      func(win uintptr)
	boxNew       func(orient, spacing int32) uintptr
	boxAppend    func(box, child uintptr)
	marginTop    func(w uintptr, m int32)
	marginBottom func(w uintptr, m int32)
	marginStart  func(w uintptr, m int32)
	marginEnd    func(w uintptr, m int32)
	hexpand      func(w uintptr, expand int32)
	vexpand      func(w uintptr, expand int32)
	grabFocus    func(w uintptr) int32
	labelNew     func(text *byte) uintptr
	labelWrap    func(label uintptr, wrap int32)
	buttonNew    func(text *byte) uintptr
	entryNew     func() uintptr
	setText      func(editable uintptr, text *byte)
	getText      func(editable uintptr) uintptr
	listNew      func() uintptr
	listInsert   func(box, child uintptr, pos int32)
	listSelected func(box uintptr) uintptr
	listSelect   func(box, row uintptr)
	rowAt        func(box uintptr, index int32) uintptr
	rowIndex     func(row uintptr) int32
	scrollNew    func() uintptr
	scrollChild  func(scroll, child uintptr)
	scrollMin    func(scroll uintptr, height int32)
	scrollPolicy func(scroll uintptr, h, v int32)
	self         func() uintptr
}

var (
	mu      sync.Mutex
	bound   symbols
	loaded  bool
	ready   bool
	ownerID uint64
	gtkLib  uintptr
	jobs    = make(chan func(), 32)

	x11Once       sync.Once
	x11Err        error
	realize       func(uintptr)
	nativeSurface func(uintptr) uintptr
	x11XID        func(uintptr) uintptr
)

// Available reports whether GTK 4 and GLib can be loaded.
// ctx is the caller context used to resolve the toolkit through the exec driver.
func Available(ctx context.Context) error {
	if err := native.Prepare(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if err := bindOnce(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

// SetPrgname sets the GLib program name used as the default window class.
// A missing library leaves the process name unchanged.
// This opens glib only. Loading GTK 4 here would abort a later WebKit2GTK 4.1
// process, which is GTK 3.
func SetPrgname(name string) {
	if name == "" {
		return
	}
	glib, err := native.OpenChain(native.Global|native.Lazy, "libglib-2.0.so.0")
	if err != nil {
		return
	}
	got, err := native.Bind[Glib](glib)
	if err != nil {
		return
	}
	withCString(name, func(p *byte) { got.GSetPrgname(p) })
}

// Owned reports whether Ensure has claimed a thread.
func Owned() bool {
	mu.Lock()
	defer mu.Unlock()
	return ready
}

// OnOwner reports whether this OS thread claimed GTK.
func OnOwner() bool {
	id := osThread()
	mu.Lock()
	defer mu.Unlock()
	return ready && id != 0 && ownerID == id
}

// Ensure initializes GTK on this thread and claims the main context.
// The caller keeps this OS thread locked.
func Ensure() error {
	if err := bindOnce(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	id := osThread()
	if id == 0 {
		return fmt.Errorf("%w: thread id", ErrUnavailable)
	}
	mu.Lock()
	if ready {
		mine := ownerID == id
		mu.Unlock()
		if !mine {
			return fmt.Errorf("%w: gtk is owned by another thread", ErrUnavailable)
		}
		return nil
	}
	mu.Unlock()
	if bound.initCheck() == 0 {
		return fmt.Errorf("%w: gtk_init_check failed", ErrUnavailable)
	}
	mu.Lock()
	if ready && ownerID != id {
		mu.Unlock()
		return fmt.Errorf("%w: gtk is owned by another thread", ErrUnavailable)
	}
	ownerID = id
	ready = true
	mu.Unlock()
	return nil
}

// Do runs fn on the owner thread.
// The owner runs fn immediately. Another thread waits until Poll runs it.
func Do(fn func()) error {
	if fn == nil {
		return nil
	}
	if OnOwner() {
		fn()
		return nil
	}
	if !Owned() {
		return fmt.Errorf("%w: gtk is not started", ErrUnavailable)
	}
	done := make(chan struct{})
	jobs <- func() {
		fn()
		close(done)
	}
	<-done
	return nil
}

// Enqueue runs fn on the owner thread without waiting.
func Enqueue(fn func()) error {
	if fn == nil {
		return nil
	}
	if !Owned() {
		return fmt.Errorf("%w: gtk is not started", ErrUnavailable)
	}
	jobs <- fn
	return nil
}

// Poll runs queued work and one non-blocking main-context iteration.
// A thread that does not own GTK returns 0.
func Poll() int32 {
	if !OnOwner() {
		return 0
	}
	var dispatched int32
	for {
		select {
		case fn := <-jobs:
			fn()
			dispatched = 1
		default:
			if bound.iterate(0, 0) != 0 {
				return 1
			}
			return dispatched
		}
	}
}

// blockGTK4 reports that loading libgtk-4 would mix majors.
// GTK 3 already mapped, or the only WebKit is WebKit2GTK 4.1.
// WebKitGTK 6 keeps this process on GTK 4.
func blockGTK4() bool {
	return blockGTK4Choice(
		native.Loaded("libgtk-3.so.0"),
		sonamePresent("libwebkitgtk-6.0.so.4"),
		sonamePresent("libwebkit2gtk-4.1.so.0"),
	)
}

func blockGTK4Choice(gtk3Loaded, webkit6, webkit41 bool) bool {
	if gtk3Loaded {
		return true
	}
	if webkit6 {
		return false
	}
	return webkit41
}

func sonamePresent(name string) bool {
	for _, dir := range native.SearchDirs() {
		info, err := os.Stat(filepath.Join(dir, name))
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func bindOnce() error {
	mu.Lock()
	defer mu.Unlock()
	if loaded {
		return nil
	}
	if blockGTK4() {
		return errMixedGTK
	}
	glib, err := native.OpenChain(native.Global|native.Lazy, "libglib-2.0.so.0")
	if err != nil {
		return err
	}
	gobject, err := native.OpenChain(native.Global|native.Lazy, "libgobject-2.0.so.0")
	if err != nil {
		return err
	}
	gtk, err := native.OpenChain(native.Global|native.Lazy, "libgtk-4.so.1")
	if err != nil {
		return err
	}
	gtkLib = gtk
	libc, err := native.OpenChain(native.Lazy, "libc.so.6")
	if err != nil {
		return err
	}
	glibSyms, err := native.Bind[Glib](glib)
	if err != nil {
		return err
	}
	objSyms, err := native.Bind[GObject](gobject)
	if err != nil {
		return err
	}
	gtkSyms, err := native.Bind[Gtk](gtk)
	if err != nil {
		return err
	}
	libcSyms, err := native.Bind[Libc](libc)
	if err != nil {
		return err
	}
	bound.setPrgname = glibSyms.GSetPrgname
	bound.iterate = glibSyms.GMainContextIteration
	bound.signal = objSyms.GSignalConnectData
	bound.initCheck = gtkSyms.GtkInitCheck
	bound.windowNew = gtkSyms.GtkWindowNew
	bound.setTitle = gtkSyms.GtkWindowSetTitle
	bound.setSize = gtkSyms.GtkWindowSetDefaultSize
	bound.setModal = gtkSyms.GtkWindowSetModal
	bound.setChild = gtkSyms.GtkWindowSetChild
	bound.setResizable = gtkSyms.GtkWindowSetResizable
	bound.setDefault = gtkSyms.GtkWindowSetDefaultWidget
	bound.present = gtkSyms.GtkWindowPresent
	bound.destroy = gtkSyms.GtkWindowDestroy
	bound.boxNew = gtkSyms.GtkBoxNew
	bound.boxAppend = gtkSyms.GtkBoxAppend
	bound.marginTop = gtkSyms.GtkWidgetSetMarginTop
	bound.marginBottom = gtkSyms.GtkWidgetSetMarginBottom
	bound.marginStart = gtkSyms.GtkWidgetSetMarginStart
	bound.marginEnd = gtkSyms.GtkWidgetSetMarginEnd
	bound.hexpand = gtkSyms.GtkWidgetSetHexpand
	bound.vexpand = gtkSyms.GtkWidgetSetVexpand
	bound.grabFocus = gtkSyms.GtkWidgetGrabFocus
	bound.labelNew = gtkSyms.GtkLabelNew
	bound.labelWrap = gtkSyms.GtkLabelSetWrap
	bound.buttonNew = gtkSyms.GtkButtonNewWithLabel
	bound.entryNew = gtkSyms.GtkEntryNew
	bound.setText = gtkSyms.GtkEditableSetText
	bound.getText = gtkSyms.GtkEditableGetText
	bound.listNew = gtkSyms.GtkListBoxNew
	bound.listInsert = gtkSyms.GtkListBoxInsert
	bound.listSelected = gtkSyms.GtkListBoxGetSelectedRow
	bound.listSelect = gtkSyms.GtkListBoxSelectRow
	bound.rowAt = gtkSyms.GtkListBoxGetRowAtIndex
	bound.rowIndex = gtkSyms.GtkListBoxRowGetIndex
	bound.scrollNew = gtkSyms.GtkScrolledWindowNew
	bound.scrollChild = gtkSyms.GtkScrolledWindowSetChild
	bound.scrollMin = gtkSyms.GtkScrolledWindowSetMinContentHeight
	bound.scrollPolicy = gtkSyms.GtkScrolledWindowSetPolicy
	bound.self = libcSyms.PthreadSelf
	loaded = true
	return nil
}

// SurfaceXID is the X11 window id for a realized GTK widget.
// Wayland and a missing GDK X11 backend return ErrUnavailable.
func SurfaceXID(widget uintptr) (uint32, error) {
	if widget == 0 {
		return 0, ErrUnavailable
	}
	if err := bindX11(); err != nil {
		return 0, err
	}
	var xid uint32
	var callErr error
	run := func() {
		realize(widget)
		surface := nativeSurface(widget)
		if surface == 0 {
			callErr = ErrUnavailable
			return
		}
		id := x11XID(surface)
		if id == 0 {
			callErr = ErrUnavailable
			return
		}
		xid = uint32(id)
	}
	if OnOwner() {
		run()
		return xid, callErr
	}
	if err := Do(run); err != nil {
		return 0, err
	}
	return xid, callErr
}

func bindX11() error {
	x11Once.Do(func() {
		if err := bindOnce(); err != nil {
			x11Err = err
			return
		}
		if gtkLib == 0 {
			x11Err = ErrUnavailable
			return
		}
		x11Syms, err := native.Bind[GDKX11](gtkLib)
		if err != nil {
			x11Err = err
			return
		}
		realize = x11Syms.GtkWidgetRealize
		nativeSurface = x11Syms.GtkNativeGetSurface
		x11XID = x11Syms.GdkX11SurfaceGetXid
	})
	return x11Err
}

func osThread() uint64 {
	if err := bindOnce(); err != nil || bound.self == nil {
		return 0
	}
	return uint64(bound.self())
}

func withCString(s string, fn func(*byte)) {
	b := native.CString(s)
	fn(cStringPointer(b))
	runtime.KeepAlive(b)
}

func cStringPointer(text []byte) *byte {
	if len(text) == 0 {
		return nil
	}
	return &text[0]
}

func connect(obj uintptr, signal string, handler uintptr) {
	name := native.CString(signal)
	bound.signal(obj, cStringPointer(name), handler, 0, 0, 0)
	runtime.KeepAlive(name)
}

func margin(widget uintptr, px int32) {
	bound.marginTop(widget, px)
	bound.marginBottom(widget, px)
	bound.marginStart(widget, px)
	bound.marginEnd(widget, px)
}

func label(text string) uintptr {
	var widget uintptr
	withCString(text, func(p *byte) { widget = bound.labelNew(p) })
	bound.labelWrap(widget, 1)
	bound.hexpand(widget, 1)
	return widget
}

func button(text string) uintptr {
	var widget uintptr
	withCString(text, func(p *byte) { widget = bound.buttonNew(p) })
	connect(widget, "clicked", clickCB)
	return widget
}
