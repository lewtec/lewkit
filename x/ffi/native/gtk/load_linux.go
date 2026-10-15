//go:build linux && !android

package gtk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/ffi/native"
)

// ErrUnavailable means GTK 4 is missing or this process cannot open a display.
var ErrUnavailable = errs.Wrap(errs.ErrUnavailable, "gtk unavailable")

// errMixedGTK means this process is on GTK 3, so GTK 4 must not be loaded.
var errMixedGTK = errors.New("GTK 4 cannot share a process with GTK 3")

const (
	orientVertical   = 1
	policyNever      = 2
	policyAutomatic  = 1
	listBoxAppendPos = -1
)

var (
	mu      sync.Mutex
	ready   bool
	ownerID uint64
	jobs    = make(chan func(), 32)
)

// gtkBound is glib, gobject, GTK, and libc from one load.
type gtkBound struct {
	glib    Glib
	gobject GObject
	widgets Gtk
	threads Libc
}

var (
	bindGlib    = native.BindOnce[Glib](native.Global|native.Lazy, "libglib-2.0.so.0")
	bindGObject = native.BindOnce[GObject](native.Global|native.Lazy, "libgobject-2.0.so.0")
	bindGTK     = native.BindOnce[Gtk](native.Global|native.Lazy, "libgtk-4.so.1")
	bindThreads = native.BindOnce[Libc](native.Lazy, "libc.so.6")
	bindGDK     = native.BindOnce[GDKX11](native.Global|native.Lazy, "libgtk-4.so.1")
)

// loadGTK binds the GTK libraries and returns them from that same call.
// The load stays on the caller. gtk_init still runs later, on the owner thread.
// bindGTK is not called when this process must stay on GTK 3.
func loadGTK() (gtkBound, error) {
	if blockGTK4() {
		return gtkBound{}, errMixedGTK
	}
	glib, err := bindGlib()
	if err != nil {
		return gtkBound{}, err
	}
	gobject, err := bindGObject()
	if err != nil {
		return gtkBound{}, err
	}
	widgets, err := bindGTK()
	if err != nil {
		return gtkBound{}, err
	}
	threads, err := bindThreads()
	if err != nil {
		return gtkBound{}, err
	}
	return gtkBound{glib: glib, gobject: gobject, widgets: widgets, threads: threads}, nil
}

// loadGDK binds the GDK X11 symbols after GTK itself loaded.
func loadGDK() (GDKX11, error) {
	if _, err := loadGTK(); err != nil {
		return GDKX11{}, err
	}
	return bindGDK()
}

// Available reports whether GTK 4 and GLib can be loaded.
// ctx is the caller context used to resolve the toolkit through the exec driver.
func Available(ctx context.Context) error {
	if err := native.Prepare(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if _, err := loadGTK(); err != nil {
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
	got, err := bindGlib()
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
	bound, err := loadGTK()
	if err != nil {
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
	if bound.widgets.GtkInitCheck() == 0 {
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
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	var dispatched int32
	for {
		select {
		case fn := <-jobs:
			fn()
			dispatched = 1
		default:
			if bound.glib.GMainContextIteration(0, 0) != 0 {
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

// SurfaceXID is the X11 window id for a realized GTK widget.
// Wayland and a missing GDK X11 backend return ErrUnavailable.
func SurfaceXID(widget uintptr) (uint32, error) {
	if widget == 0 {
		return 0, ErrUnavailable
	}
	gdk, err := loadGDK()
	if err != nil {
		return 0, err
	}
	var xid uint32
	var callErr error
	run := func() {
		gdk.GtkWidgetRealize(widget)
		surface := gdk.GtkNativeGetSurface(widget)
		if surface == 0 {
			callErr = ErrUnavailable
			return
		}
		id := gdk.GdkX11SurfaceGetXid(surface)
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

func osThread() uint64 {
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	return uint64(bound.threads.PthreadSelf())
}

func withCString(s string, fn func(*byte)) {
	b := native.CString(s)
	fn(native.Data(b))
	runtime.KeepAlive(b)
}

func connect(obj uintptr, signal string, handler uintptr) {
	bound, err := loadGTK()
	if err != nil {
		return
	}
	name := native.CString(signal)
	bound.gobject.GSignalConnectData(obj, native.Data(name), handler, 0, 0, 0)
	runtime.KeepAlive(name)
}

func margin(widget uintptr, px int32) {
	bound, err := loadGTK()
	if err != nil {
		return
	}
	w := bound.widgets
	w.GtkWidgetSetMarginTop(widget, px)
	w.GtkWidgetSetMarginBottom(widget, px)
	w.GtkWidgetSetMarginStart(widget, px)
	w.GtkWidgetSetMarginEnd(widget, px)
}

func label(text string) uintptr {
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	w := bound.widgets
	var widget uintptr
	withCString(text, func(p *byte) { widget = w.GtkLabelNew(p) })
	w.GtkLabelSetWrap(widget, 1)
	w.GtkWidgetSetHexpand(widget, 1)
	return widget
}

func button(text string) uintptr {
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	var widget uintptr
	withCString(text, func(p *byte) { widget = bound.widgets.GtkButtonNewWithLabel(p) })
	connect(widget, "clicked", clickCB)
	return widget
}
