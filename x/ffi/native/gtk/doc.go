// Package gtk loads GTK 4 and GLib without cgo.
//
// Dialogs and the main context run on the thread that calls Ensure.
// Poll drains queued work and, on that thread, the default GLib context.
// A missing library or display is ErrUnavailable.
package gtk

//go:generate go run ../../../../cmd/lewkit generate header --package gtk --bind Glib --out glib_bind.go glib.h
//go:generate go run ../../../../cmd/lewkit generate header --package gtk --bind GObject --out gobject_bind.go gobject.h
//go:generate go run ../../../../cmd/lewkit generate header --package gtk --bind Gtk --out gtk_bind.go gtk.h
//go:generate go run ../../../../cmd/lewkit generate header --package gtk --bind Libc --out libc_bind.go libc.h
//go:generate go run ../../../../cmd/lewkit generate header --package gtk --bind GDKX11 --out gdkx11_bind.go gdkx11.h
