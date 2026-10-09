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
			bound.boxAppend(root, label(head))
		}
		bound.boxAppend(root, label(message))
		ok := button("OK")
		f.accept = ok
		bound.boxAppend(root, ok)
		f.win = window(title, root, 420, 160, false)
		bound.setDefault(f.win, ok)
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
		bound.boxAppend(root, label(message))
		row := bound.boxNew(0, 8)
		yes := button("Yes")
		no := button("No")
		f.accept = yes
		bound.boxAppend(row, yes)
		bound.boxAppend(row, no)
		bound.boxAppend(root, row)
		f.win = window("Confirm", root, 420, 140, false)
		bound.setDefault(f.win, yes)
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
		bound.boxAppend(root, label(prompt))
		f.entry = bound.entryNew()
		bound.hexpand(f.entry, 1)
		connect(f.entry, "activate", activeCB)
		bound.boxAppend(root, f.entry)
		row := bound.boxNew(0, 8)
		accept := button("OK")
		f.accept = accept
		bound.boxAppend(row, accept)
		bound.boxAppend(row, button("Cancel"))
		bound.boxAppend(root, row)
		f.win = window(prompt, root, 420, 160, false)
		bound.setDefault(f.win, accept)
		bound.grabFocus(f.entry)
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
		bound.boxAppend(root, label(prompt))
		f.list = bound.listNew()
		connect(f.list, "row-activated", rowCB)
		for _, item := range labels {
			bound.listInsert(f.list, label(item), listBoxAppendPos)
		}
		scroll := bound.scrollNew()
		bound.scrollChild(scroll, f.list)
		bound.scrollMin(scroll, 200)
		bound.scrollPolicy(scroll, policyNever, policyAutomatic)
		bound.vexpand(scroll, 1)
		bound.hexpand(scroll, 1)
		bound.boxAppend(root, scroll)
		row := bound.boxNew(0, 8)
		accept := button("OK")
		f.accept = accept
		bound.boxAppend(row, accept)
		bound.boxAppend(row, button("Cancel"))
		bound.boxAppend(root, row)
		title := prompt
		if title == "" {
			title = "Choose"
		}
		f.win = window(title, root, 420, 360, true)
		bound.setDefault(f.win, accept)
		if first := bound.rowAt(f.list, 0); first != 0 {
			bound.listSelect(f.list, first)
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
	bound.setModal(f.win, 1)
	connect(f.win, "close-request", closeCB)
	bound.present(f.win)
	for !f.done {
		if ctx != nil && ctx.Err() != nil {
			f.ok = false
			break
		}
		if Poll() == 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}
	bound.destroy(f.win)
	current = prev
	return nil
}

func column() uintptr {
	box := bound.boxNew(orientVertical, 12)
	margin(box, 12)
	return box
}

func window(title string, child uintptr, width, height int32, resizable bool) uintptr {
	win := bound.windowNew()
	withCString(title, func(p *byte) { bound.setTitle(win, p) })
	bound.setSize(win, width, height)
	resize := int32(0)
	if resizable {
		resize = 1
	}
	bound.setResizable(win, resize)
	bound.setChild(win, child)
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
	current.index = int(bound.rowIndex(row))
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
		f.text = native.GoString(bound.getText(f.entry))
	}
	if f.list != 0 {
		row := bound.listSelected(f.list)
		if row == 0 {
			f.ok = false
			return
		}
		f.index = int(bound.rowIndex(row))
	}
}
