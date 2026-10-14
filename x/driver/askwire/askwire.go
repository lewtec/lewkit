// Package askwire is the text a mobile dialog returns to Go.
// The first line is the status. The rest is the payload.
package askwire

import "strings"

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum askwire.go

// Status is the first line of a dialog reply.
//
//lewkit:enum
type Status string

const (
	// StatusOK is a dismissed alert, a prompt, or a chosen row.
	StatusOK Status = "ok"
	// StatusYes is a confirm accepted.
	StatusYes Status = "yes"
	// StatusNo is a confirm declined.
	StatusNo Status = "no"
	// StatusCanceled is a dialog the user dismissed.
	StatusCanceled Status = "canceled"
	// StatusNoActivity means the host has no window to present on.
	StatusNoActivity Status = "no activity"
	// StatusBusy means a dialog is already open.
	StatusBusy Status = "busy"
)

// Format joins status and payload with one newline.
func Format(status Status, payload string) string {
	return string(status) + "\n" + payload
}

// Split returns the status line and the payload after it.
func Split(raw string) (status Status, payload string) {
	line, payload, _ := strings.Cut(raw, "\n")
	return Status(line), payload
}
