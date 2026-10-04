// Package askwire is the text a mobile dialog returns to Go.
// The first line is the status. The rest is the payload.
package askwire

import "strings"

const (
	// StatusOK is a dismissed alert, a prompt, or a chosen row.
	StatusOK = "ok"
	// StatusYes is a confirm accepted.
	StatusYes = "yes"
	// StatusNo is a confirm declined.
	StatusNo = "no"
	// StatusCanceled is a dialog the user dismissed.
	StatusCanceled = "canceled"
	// StatusNoActivity means the host has no window to present on.
	StatusNoActivity = "no activity"
	// StatusBusy means a dialog is already open.
	StatusBusy = "busy"
)

// Format joins status and payload with one newline.
func Format(status, payload string) string {
	return status + "\n" + payload
}

// Split returns the status line and the payload after it.
func Split(raw string) (status, payload string) {
	status, payload, _ = strings.Cut(raw, "\n")
	return status, payload
}
