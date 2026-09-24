//go:build darwin

package cocoa

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/window"
)

const nsDragOperationCopy = 1

var (
	dropOnce  sync.Once
	dropClass objc.Class
	dropErr   error
	dropViews sync.Map // view pointer → *win

	selDraggingPasteboard = objc.RegisterName("draggingPasteboard")
	selReadObjects        = objc.RegisterName("readObjectsForClasses:options:")
	selArrayWithObject    = objc.RegisterName("arrayWithObject:")
	selArrayByAdding      = objc.RegisterName("arrayByAddingObject:")
	selAvailableType      = objc.RegisterName("availableTypeFromArray:")
	selPropertyList       = objc.RegisterName("propertyListForType:")
	selCount              = objc.RegisterName("count")
	selObjectAtIndex      = objc.RegisterName("objectAtIndex:")
	selIsFileURL          = objc.RegisterName("isFileURL")
	selPath               = objc.RegisterName("path")
	selUTF8               = objc.RegisterName("UTF8String")
	selRegisterDrag       = objc.RegisterName("registerForDraggedTypes:")
	selSetContentView     = objc.RegisterName("setContentView:")
	selInitWithFrame      = objc.RegisterName("initWithFrame:")
	selAllocView          = objc.RegisterName("alloc")
)

func ensureDropClass() error {
	dropOnce.Do(func() {
		var protocols []*objc.Protocol
		if protocol := objc.GetProtocol("NSDraggingDestination"); protocol != nil {
			protocols = []*objc.Protocol{protocol}
		}
		dropClass, dropErr = objc.RegisterClass(
			"LewkitDropView",
			objc.GetClass("NSView"),
			protocols,
			nil,
			[]objc.MethodDef{
				{Cmd: objc.RegisterName("draggingEntered:"), Fn: dropEntered},
				{Cmd: objc.RegisterName("draggingUpdated:"), Fn: dropEntered},
				{Cmd: objc.RegisterName("prepareForDragOperation:"), Fn: dropPrepare},
				{Cmd: objc.RegisterName("performDragOperation:"), Fn: dropPerform},
			},
		)
	})
	return dropErr
}

func (w *win) installDrop(width, height int) error {
	if err := ensureDropClass(); err != nil {
		return err
	}
	rect := nsRect{Size: nsSize{Width: float64(width), Height: float64(height)}}
	view := objc.ID(dropClass).Send(selAllocView)
	view = view.Send(selInitWithFrame, rect)
	if view == 0 {
		return window.ErrInit
	}
	view.Send(selSetWantsLayer, true)
	fileURL := nsstr("public.file-url")
	names := objc.ID(objc.GetClass("NSArray")).Send(selArrayWithObject, fileURL)
	names = names.Send(selArrayByAdding, nsstr("NSFilenamesPboardType"))
	view.Send(selRegisterDrag, names)
	w.wnd.Send(selSetContentView, view)
	dropViews.Store(uintptr(view), w)
	view.Send(selRelease)
	return nil
}

func dropEntered(_ objc.ID, _ objc.SEL, sender objc.ID) int {
	if !dragHasFiles(sender) {
		return 0
	}
	return nsDragOperationCopy
}

func dropPrepare(_ objc.ID, _ objc.SEL, _ objc.ID) bool { return true }

func dropPerform(self objc.ID, _ objc.SEL, sender objc.ID) bool {
	value, ok := dropViews.Load(uintptr(self))
	if !ok {
		return false
	}
	paths := dragPaths(sender)
	if len(paths) == 0 {
		return false
	}
	value.(*win).Emit(window.Drop{Paths: paths})
	return true
}

func dragHasFiles(sender objc.ID) bool {
	board := draggingBoard(sender)
	if board == 0 {
		return false
	}
	names := objc.ID(objc.GetClass("NSArray")).Send(selArrayWithObject, nsstr("public.file-url"))
	names = names.Send(selArrayByAdding, nsstr("NSFilenamesPboardType"))
	return board.Send(selAvailableType, names) != 0
}

func dragPaths(sender objc.ID) []string {
	board := draggingBoard(sender)
	if board == 0 {
		return nil
	}
	classes := objc.ID(objc.GetClass("NSArray")).Send(selArrayWithObject, objc.ID(objc.GetClass("NSURL")))
	items := objectStrings(board.Send(selReadObjects, classes, objc.ID(0)), true)
	if len(items) > 0 {
		return dropPaths(items)
	}
	listed := board.Send(selPropertyList, nsstr("NSFilenamesPboardType"))
	return dropPaths(objectStrings(listed, false))
}

func draggingBoard(sender objc.ID) objc.ID {
	if sender == 0 {
		return 0
	}
	return sender.Send(selDraggingPasteboard)
}

func objectStrings(list objc.ID, fileURL bool) []string {
	if list == 0 {
		return nil
	}
	count := int(list.Send(selCount))
	var items []string
	for index := 0; index < count; index++ {
		object := list.Send(selObjectAtIndex, index)
		if object == 0 {
			continue
		}
		if fileURL {
			if object.Send(selIsFileURL) == 0 {
				continue
			}
			object = object.Send(selPath)
		}
		if text := nsGoString(object); text != "" {
			items = append(items, text)
		}
	}
	return items
}

func nsGoString(object objc.ID) string {
	if object == 0 {
		return ""
	}
	pointer := object.Send(selUTF8)
	if pointer == 0 {
		return ""
	}
	size := cstrlen(uintptr(pointer))
	if size == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(pointer)), size))
}
