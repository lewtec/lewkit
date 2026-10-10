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
		f := &form{}
		root := column()
		if head := heading(style); head != "" {
			widgets.GtkBoxAppend(root, label(head))
		}
		widgets.GtkBoxAppend(root, label(message))
		ok := button("OK")
		f.accept = ok
		widgets.GtkBoxAppend(root, ok)
		f.win = window(title, root, 420, 160, false)
		widgets.GtkWindowSetDefaultWidget(f.win, ok)
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
		f := &form{}
		root := column()
		widgets.GtkBoxAppend(root, label(message))
		row := widgets.GtkBoxNew(0, 8)
		yes := button("Yes")
		no := button("No")
		f.accept = yes
		widgets.GtkBoxAppend(row, yes)
		widgets.GtkBoxAppend(row, no)
		widgets.GtkBoxAppend(root, row)
		f.win = window("Confirm", root, 420, 140, false)
		widgets.GtkWindowSetDefaultWidget(f.win, yes)
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
		f := &form{}
		root := column()
		widgets.GtkBoxAppend(root, label(prompt))
		f.entry = widgets.GtkEntryNew()
		widgets.GtkWidgetSetHexpand(f.entry, 1)
		connect(f.entry, "activate", activeCB)
		widgets.GtkBoxAppend(root, f.entry)
		row := widgets.GtkBoxNew(0, 8)
		accept := button("OK")
		f.accept = accept
		widgets.GtkBoxAppend(row, accept)
		widgets.GtkBoxAppend(row, button("Cancel"))
		widgets.GtkBoxAppend(root, row)
		f.win = window(prompt, root, 420, 160, false)
		widgets.GtkWindowSetDefaultWidget(f.win, accept)
		widgets.GtkWidgetGrabFocus(f.entry)
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
		f := &form{index: -1}
		root := column()
		widgets.GtkBoxAppend(root, label(prompt))
		f.list = widgets.GtkListBoxNew()
		connect(f.list, "row-activated", rowCB)
		for _, item := range labels {
			widgets.GtkListBoxInsert(f.list, label(item), listBoxAppendPos)
		}
		scroll := widgets.GtkScrolledWindowNew()
		widgets.GtkScrolledWindowSetChild(scroll, f.list)
		widgets.GtkScrolledWindowSetMinContentHeight(scroll, 200)
		widgets.GtkScrolledWindowSetPolicy(scroll, policyNever, policyAutomatic)
		widgets.GtkWidgetSetVexpand(scroll, 1)
		widgets.GtkWidgetSetHexpand(scroll, 1)
		widgets.GtkBoxAppend(root, scroll)
		row := widgets.GtkBoxNew(0, 8)
		accept := button("OK")
		f.accept = accept
		widgets.GtkBoxAppend(row, accept)
		widgets.GtkBoxAppend(row, button("Cancel"))
		widgets.GtkBoxAppend(root, row)
		title := prompt
		if title == "" {
			title = "Choose"
		}
		f.win = window(title, root, 420, 360, true)
		widgets.GtkWindowSetDefaultWidget(f.win, accept)
		if first := widgets.GtkListBoxGetRowAtIndex(f.list, 0); first != 0 {
			widgets.GtkListBoxSelectRow(f.list, first)
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
	prev := current
	current = f
	widgets.GtkWindowSetModal(f.win, 1)
	connect(f.win, "close-request", closeCB)
	widgets.GtkWindowPresent(f.win)
	for !f.done {
		if ctx.Err() != nil {
			f.ok = false
			break
		}
		if Poll() == 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	widgets.GtkWindowDestroy(f.win)
	current = prev
	return nil
}

func column() uintptr {
	box := widgets.GtkBoxNew(orientVertical, 12)
	margin(box, 12)
	return box
}

func window(title string, child uintptr, width, height int32, resizable bool) uintptr {
	win := widgets.GtkWindowNew()
	withCString(title, func(p *byte) { widgets.GtkWindowSetTitle(win, p) })
	widgets.GtkWindowSetDefaultSize(win, width, height)
	resize := int32(0)
	if resizable {
		resize = 1
	}
	widgets.GtkWindowSetResizable(win, resize)
	widgets.GtkWindowSetChild(win, child)
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
	current.ok = true
	current.index = int(widgets.GtkListBoxRowGetIndex(row))
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
	f.ok = true
	if f.entry != 0 {
		f.text = native.GoString(widgets.GtkEditableGetText(f.entry))
	}
	if f.list != 0 {
		row := widgets.GtkListBoxGetSelectedRow(f.list)
		if row == 0 {
			f.ok = false
			return
		}
		f.index = int(widgets.GtkListBoxRowGetIndex(row))
	}
}
