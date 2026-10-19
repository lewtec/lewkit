package uikitapp

import (
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver/askwire"
)

// Kind values match hostask: alert, confirm, prompt, choose.
const (
	kindAlert   = 1
	kindConfirm = 2
	kindPrompt  = 3
	kindChoose  = 4
)

// UIAlertControllerStyle. A text field is legal only on an alert.
const (
	styleActionSheet = 0
	styleAlert       = 1
)

// askStyle matches the Swift host: choose is an action sheet, and a prompt is an alert.
func askStyle(kind int) uintptr {
	if kind == kindChoose {
		return styleActionSheet
	}
	return styleAlert
}

func askTitle(kind int, title string) string {
	if strings.TrimSpace(title) != "" {
		return title
	}
	switch kind {
	case kindConfirm:
		return "Confirm"
	case kindPrompt:
		return "Input"
	case kindChoose:
		return "Choose"
	default:
		return "Error"
	}
}

func askLines(body string) []string {
	var out []string
	for line := range strings.SplitSeq(body, "\n") {
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func safeName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == ".." || base == string(filepath.Separator) {
		return "untitled"
	}
	return base
}

func batteryRead(state int, level float64) (string, int) {
	status := "Unknown"
	switch state {
	case 1:
		status = "Discharging"
	case 2:
		status = "Charging"
	case 3:
		status = "Full"
	}
	if level < 0 {
		return status, -1
	}
	percent := int(level*100 + 0.5)
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		return status, -1
	}
	return status, percent
}

func appearanceName(style int) string {
	if style == 2 {
		return "dark"
	}
	return "light"
}

func okText() string { return askwire.Format(askwire.StatusOK, "") }

func okBody(body string) string { return askwire.Format(askwire.StatusOK, body) }

func yesText() string { return askwire.Format(askwire.StatusYes, "") }

func noText() string { return askwire.Format(askwire.StatusNo, "") }

func cancelText() string { return askwire.Format(askwire.StatusCanceled, "") }

func busyText() string { return askwire.Format(askwire.StatusBusy, "") }

func idleText() string { return askwire.Format(askwire.StatusNoActivity, "") }

func failText(msg string) string { return "error\n" + msg }
