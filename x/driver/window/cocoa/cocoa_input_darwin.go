//go:build darwin

package cocoa

import (
	"image"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	nsLeftMouseDown     = 1
	nsLeftMouseUp       = 2
	nsRightMouseDown    = 3
	nsRightMouseUp      = 4
	nsMouseMoved        = 5
	nsLeftMouseDragged  = 6
	nsRightMouseDragged = 7
	nsKeyDown           = 10
	nsKeyUp             = 11
	nsScrollWheel       = 22
	nsOtherMouseDown    = 25
	nsOtherMouseUp      = 26
	nsOtherMouseDragged = 27
	nsModShift          = 1 << 17
	nsModCtrl           = 1 << 18
	nsModAlt            = 1 << 19
	nsModSuper          = 1 << 20
)

var (
	selType                 = objc.RegisterName("type")
	selWindow               = objc.RegisterName("window")
	selLocationInWindow     = objc.RegisterName("locationInWindow")
	selButtonNumber         = objc.RegisterName("buttonNumber")
	selDeltaX               = objc.RegisterName("deltaX")
	selDeltaY               = objc.RegisterName("deltaY")
	selKeyCode              = objc.RegisterName("keyCode")
	selCharacters           = objc.RegisterName("characters")
	selIsARepeat            = objc.RegisterName("isARepeat")
	selModifierFlags        = objc.RegisterName("modifierFlags")
	selPressedMouseButtons  = objc.RegisterName("pressedMouseButtons")
	selSetAcceptsMouseMoved = objc.RegisterName("setAcceptsMouseMovedEvents:")
	selUTF8String           = objc.RegisterName("UTF8String")
	locationFn              func(objc.ID, objc.SEL) nsPoint
	deltaFn                 func(objc.ID, objc.SEL) float64
)

func acceptMouseMoved(wnd objc.ID) {
	setBool(wnd, selSetAcceptsMouseMoved, true)
}

func dispatchInput(ev objc.ID) {
	if ev == 0 {
		return
	}
	nsw := ev.Send(selWindow)
	if nsw == 0 {
		return
	}
	var target *win
	live.Range(func(k, _ any) bool {
		w := k.(*win)
		w.mu.Lock()
		match := w.wnd == nsw
		w.mu.Unlock()
		if match {
			target = w
			return false
		}
		return true
	})
	if target == nil {
		return
	}
	typ := int(ev.Send(selType))
	switch typ {
	case nsLeftMouseDown, nsRightMouseDown, nsOtherMouseDown,
		nsLeftMouseUp, nsRightMouseUp, nsOtherMouseUp,
		nsMouseMoved, nsLeftMouseDragged, nsRightMouseDragged, nsOtherMouseDragged,
		nsScrollWheel:
		view := nsw.Send(selContentView)
		if view != 0 && view.Send(selInLiveResize) != 0 {
			return
		}
	}
	pos := target.pointerPos(ev)
	buttons := int(objc.ID(objc.GetClass("NSEvent")).Send(selPressedMouseButtons))
	switch typ {
	case nsLeftMouseDown, nsRightMouseDown, nsOtherMouseDown:
		target.Emit(window.Pointer{Pos: pos, Button: cocoaButton(int(ev.Send(selButtonNumber))), Pressed: true, Buttons: buttons})
	case nsLeftMouseUp, nsRightMouseUp, nsOtherMouseUp:
		target.Emit(window.Pointer{Pos: pos, Button: cocoaButton(int(ev.Send(selButtonNumber))), Pressed: false, Buttons: buttons})
	case nsMouseMoved, nsLeftMouseDragged, nsRightMouseDragged, nsOtherMouseDragged:
		target.Emit(window.Pointer{Pos: pos, Buttons: buttons})
	case nsScrollWheel:
		dx, dy := eventDelta(ev, selDeltaX), eventDelta(ev, selDeltaY)
		scale := target.scale()
		target.Emit(window.Scroll{
			Pos:   pos,
			Delta: image.Pt(int(dx*scale+0.5), int(-dy*scale+0.5)),
		})
	case nsKeyDown, nsKeyUp:
		target.Emit(window.Key{
			Rune:    firstRune(ev.Send(selCharacters)),
			Code:    uint32(ev.Send(selKeyCode)),
			Pressed: typ == nsKeyDown,
			Repeat:  ev.Send(selIsARepeat) != 0,
			Mod:     cocoaMod(uintptr(ev.Send(selModifierFlags))),
		})
	}
}

func (w *win) pointerPos(ev objc.ID) image.Point {
	loc := eventLocation(ev)
	w.mu.Lock()
	wnd := w.wnd
	w.mu.Unlock()
	if wnd == 0 {
		return image.Point{}
	}
	view := wnd.Send(selContentView)
	rect := boundsOf(view)
	scale := w.scale()
	if scale < 1 {
		scale = 1
	}
	x := int(loc.X*scale + 0.5)
	y := int((rect.Size.Height-loc.Y)*scale + 0.5)
	return image.Pt(x, y)
}

func cocoaButton(n int) int {
	switch n {
	case 0:
		return 1
	case 1:
		return 2
	default:
		return 3
	}
}

func cocoaMod(flags uintptr) window.Modifier {
	var m window.Modifier
	if flags&nsModShift != 0 {
		m |= window.ModShift
	}
	if flags&nsModCtrl != 0 {
		m |= window.ModCtrl
	}
	if flags&nsModAlt != 0 {
		m |= window.ModAlt
	}
	if flags&nsModSuper != 0 {
		m |= window.ModSuper
	}
	return m
}

func eventLocation(ev objc.ID) nsPoint {
	if locationFn == nil {
		native.Register(&locationFn, objcMsgSend)
	}
	return locationFn(ev, selLocationInWindow)
}

func eventDelta(ev objc.ID, sel objc.SEL) float64 {
	if deltaFn == nil {
		native.Register(&deltaFn, objcMsgSend)
	}
	return deltaFn(ev, sel)
}

func firstRune(ns objc.ID) rune {
	if ns == 0 {
		return 0
	}
	p := ns.Send(selUTF8String)
	if p == 0 {
		return 0
	}
	n := cstrlen(uintptr(p))
	s := string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
	for _, r := range s {
		return r
	}
	return 0
}

func cstrlen(p uintptr) int {
	n := 0
	for *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0 {
		n++
	}
	return n
}
