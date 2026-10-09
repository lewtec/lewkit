// Package gtk loads GTK 4 and GLib without cgo.
//
// Dialogs and the main context run on the thread that calls Ensure.
// Poll drains queued work and, on that thread, the default GLib context.
// A missing library or display is ErrUnavailable.
package gtk
