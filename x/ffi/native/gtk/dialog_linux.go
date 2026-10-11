//go:build linux

package gtk

import (
	"context"
	"time"

	"github.com/ebitengine/purego"
	"github.com/lewtec/lewkit/x/ffi/native"
)

type form struct {
	done   bool
	ok     bool
	text   string
	index  int
	win    uintptr
	entry  uintptr
	list   uintptr
	accept uintptr
}

var (
	current  *form
	clickCB  = purego.NewCallback(onClick)
	activeCB = purego.NewCallback(onActivate)
	rowCB    = purego.NewCallback(onRow)
	closeCB  = purego.NewCallback(onClose)
)

// Show presents one message and waits until the user dismisses it.
// style is informational, warning, or critical.
func Show(ctx context.Context, title, message, style string) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	return onOwner(func() error {
		bound, err := loadGTK()
		if err != nil {
			return err
		}
		w := bound.widgets
		f := &form{}
		root := column()
		if head := heading(style); head != "" {
			w.GtkBoxAppend(root, label(head))
		}
		w.GtkBoxAppend(root, label(message))
		ok := button("OK")
		f.accept = ok
		w.GtkBoxAppend(root, ok)
		f.win = window(title, root, 420, 160, false)
		w.GtkWindowSetDefaultWidget(f.win, ok)
		return present(ctx, f)
	})
}

// Confirm asks a yes or no question.
func Confirm(ctx context.Context, message string) (bool, error) {
	if err := ctxErr(ctx); err != nil {
		return false, err
	}
	var ok bool
	err := onOwner(func() error {
		bound, err := loadGTK()
		if err != nil {
			return err
		}
		w := bound.widgets
		f := &form{}
		root := column()
		w.GtkBoxAppend(root, label(message))
		row := w.GtkBoxNew(0, 8)
		yes := button("Yes")
		no := button("No")
		f.accept = yes
		w.GtkBoxAppend(row, yes)
		w.GtkBoxAppend(row, no)
		w.GtkBoxAppend(root, row)
		f.win = window("Confirm", root, 420, 140, false)
		w.GtkWindowSetDefaultWidget(f.win, yes)
		if err := present(ctx, f); err != nil {
			return err
		}
		ok = f.ok
		return nil
	})
	return ok, err
}

// Prompt reads one line of text.
// The second result is false when the user cancels.
func Prompt(ctx context.Context, prompt string) (string, bool, error) {
	if err := ctxErr(ctx); err != nil {
		return "", false, err
	}
	var text string
	var ok bool
	err := onOwner(func() error {
		bound, err := loadGTK()
		if err != nil {
			return err
		}
		w := bound.widgets
		f := &form{}
		root := column()
		w.GtkBoxAppend(root, label(prompt))
		f.entry = w.GtkEntryNew()
		w.GtkWidgetSetHexpand(f.entry, 1)
		connect(f.entry, "activate", activeCB)
		w.GtkBoxAppend(root, f.entry)
		row := w.GtkBoxNew(0, 8)
		accept := button("OK")
		f.accept = accept
		w.GtkBoxAppend(row, accept)
		w.GtkBoxAppend(row, button("Cancel"))
		w.GtkBoxAppend(root, row)
		f.win = window(prompt, root, 420, 160, false)
		w.GtkWindowSetDefaultWidget(f.win, accept)
		w.GtkWidgetGrabFocus(f.entry)
		if err := present(ctx, f); err != nil {
			return err
		}
		text, ok = f.text, f.ok
		return nil
	})
	return text, ok, err
}

// Choose selects one label. The index is into labels.
// The second result is false when the user cancels.
func Choose(ctx context.Context, prompt string, labels []string) (int, bool, error) {
	if err := ctxErr(ctx); err != nil {
		return 0, false, err
	}
	if len(labels) == 0 {
		return 0, false, nil
	}
	var index int
	var ok bool
	err := onOwner(func() error {
		bound, err := loadGTK()
		if err != nil {
			return err
		}
		w := bound.widgets
		f := &form{index: -1}
		root := column()
		w.GtkBoxAppend(root, label(prompt))
		f.list = w.GtkListBoxNew()
		connect(f.list, "row-activated", rowCB)
		for _, item := range labels {
			w.GtkListBoxInsert(f.list, label(item), listBoxAppendPos)
		}
		scroll := w.GtkScrolledWindowNew()
		w.GtkScrolledWindowSetChild(scroll, f.list)
		w.GtkScrolledWindowSetMinContentHeight(scroll, 200)
		w.GtkScrolledWindowSetPolicy(scroll, policyNever, policyAutomatic)
		w.GtkWidgetSetVexpand(scroll, 1)
		w.GtkWidgetSetHexpand(scroll, 1)
		w.GtkBoxAppend(root, scroll)
		row := w.GtkBoxNew(0, 8)
		accept := button("OK")
		f.accept = accept
		w.GtkBoxAppend(row, accept)
		w.GtkBoxAppend(row, button("Cancel"))
		w.GtkBoxAppend(root, row)
		title := prompt
		if title == "" {
			title = "Choose"
		}
		f.win = window(title, root, 420, 360, true)
		w.GtkWindowSetDefaultWidget(f.win, accept)
		if first := w.GtkListBoxGetRowAtIndex(f.list, 0); first != 0 {
			w.GtkListBoxSelectRow(f.list, first)
		}
		if err := present(ctx, f); err != nil {
			return err
		}
		index, ok = f.index, f.ok
		return nil
	})
	return index, ok, err
}

func onOwner(fn func() error) error {
	if OnOwner() {
		return fn()
	}
	if !Owned() {
		if err := Ensure(); err != nil {
			return err
		}
		return fn()
	}
	var err error
	if runErr := Do(func() { err = fn() }); runErr != nil {
		return runErr
	}
	return err
}

func present(ctx context.Context, f *form) error {
	bound, err := loadGTK()
	if err != nil {
		return err
	}
	w := bound.widgets
	prev := current
	current = f
	w.GtkWindowSetModal(f.win, 1)
	connect(f.win, "close-request", closeCB)
	w.GtkWindowPresent(f.win)
	for !f.done {
		if ctx.Err() != nil {
			f.ok = false
			break
		}
		if Poll() == 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	w.GtkWindowDestroy(f.win)
	current = prev
	return nil
}

func column() uintptr {
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	box := bound.widgets.GtkBoxNew(orientVertical, 12)
	margin(box, 12)
	return box
}

func window(title string, child uintptr, width, height int32, resizable bool) uintptr {
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	w := bound.widgets
	win := w.GtkWindowNew()
	withCString(title, func(p *byte) { w.GtkWindowSetTitle(win, p) })
	w.GtkWindowSetDefaultSize(win, width, height)
	resize := int32(0)
	if resizable {
		resize = 1
	}
	w.GtkWindowSetResizable(win, resize)
	w.GtkWindowSetChild(win, child)
	return win
}

func heading(style string) string {
	switch style {
	case "warning":
		return "Warning"
	case "critical":
		return "Error"
	default:
		return ""
	}
}

func ctxErr(ctx context.Context) error {
	if err := native.Prepare(ctx); err != nil {
		return err
	}
	return nil
}

func onClick(button, _ uintptr) uintptr {
	if current == nil {
		return 0
	}
	if button == current.accept {
		acceptForm(current)
	}
	current.done = true
	return 0
}

func onActivate(_, _ uintptr) uintptr {
	if current == nil {
		return 0
	}
	acceptForm(current)
	current.done = true
	return 0
}

func onRow(_, row, _ uintptr) uintptr {
	if current == nil {
		return 0
	}
	bound, err := loadGTK()
	if err != nil {
		return 0
	}
	current.ok = true
	current.index = int(bound.widgets.GtkListBoxRowGetIndex(row))
	current.done = true
	return 0
}

func onClose(_, _ uintptr) uintptr {
	if current != nil {
		current.ok = false
		current.done = true
	}
	return 1
}

func acceptForm(f *form) {
	bound, err := loadGTK()
	if err != nil {
		f.ok = false
		return
	}
	w := bound.widgets
	f.ok = true
	if f.entry != 0 {
		f.text = native.GoString(w.GtkEditableGetText(f.entry))
	}
	if f.list != 0 {
		row := w.GtkListBoxGetSelectedRow(f.list)
		if row == 0 {
			f.ok = false
			return
		}
		f.index = int(w.GtkListBoxRowGetIndex(row))
	}
}
