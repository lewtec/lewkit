//go:build ios

package uikitapp

import (
	"context"

	"github.com/ebitengine/purego/objc"
)

var (
	selAlert     = objc.RegisterName("alertControllerWithTitle:message:preferredStyle:")
	selAction    = objc.RegisterName("actionWithTitle:style:handler:")
	selAddAction = objc.RegisterName("addAction:")
	selAddField  = objc.RegisterName("addTextFieldWithConfigurationHandler:")
	selFields    = objc.RegisterName("textFields")
	selText      = objc.RegisterName("text")
	selSetText   = objc.RegisterName("setText:")
)

// Ask presents an alert, confirm, prompt, or choose.
// kind is 1 alert, 2 confirm, 3 prompt, 4 choose.
func Ask(ctx context.Context, kind int, title, body string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !lockGate() {
		return busyText(), nil
	}
	defer unlockGate()
	return wait(ctx, func(done func(string)) {
		showAsk(kind, title, body, hook(done))
	})
}

func showAsk(kind int, title, body string, finish func(string)) {
	withPool(func() {
		presenter := topController()
		if presenter == 0 {
			finish(idleText())
			return
		}
		var message objc.ID
		if kind != kindPrompt && kind != kindChoose {
			message = ns(body)
		}
		style := uintptr(0)
		if kind == kindChoose {
			style = 1
		}
		alert := cls("UIAlertController").Send(selAlert, ns(askTitle(kind, title)), message, style)
		if alert == 0 {
			finish(failText("alert"))
			return
		}
		switch kind {
		case kindConfirm:
			alert.Send(selAddAction, button("Yes", 0, func() { finish(yesText()) }))
			alert.Send(selAddAction, button("No", 1, func() { finish(noText()) }))
		case kindPrompt:
			addField(alert, body)
			alert.Send(selAddAction, button("OK", 0, func() { finish(okBody(fieldText(alert))) }))
			alert.Send(selAddAction, button("Cancel", 1, func() { finish(cancelText()) }))
		case kindChoose:
			for _, line := range askLines(body) {
				label := line
				alert.Send(selAddAction, button(label, 0, func() { finish(okBody(label)) }))
			}
			alert.Send(selAddAction, button("Cancel", 1, func() { finish(cancelText()) }))
		default:
			alert.Send(selAddAction, button("OK", 0, func() { finish(okText()) }))
		}
		present(presenter, alert)
	})
}

func button(title string, style uintptr, fn func()) objc.ID {
	block := objc.NewBlock(func(_ objc.Block, _ objc.ID) { fn() })
	item := cls("UIAlertAction").Send(selAction, ns(title), style, block)
	block.Release()
	return item
}

func addField(alert objc.ID, body string) {
	block := objc.NewBlock(func(_ objc.Block, field objc.ID) {
		if body != "" && field != 0 {
			field.Send(selSetText, ns(body))
		}
	})
	alert.Send(selAddField, block)
	block.Release()
}

func fieldText(alert objc.ID) string {
	fields := alert.Send(selFields)
	if fields == 0 {
		return ""
	}
	field := fields.Send(selFirst)
	if field == 0 {
		return ""
	}
	return objcText(field.Send(selText))
}
