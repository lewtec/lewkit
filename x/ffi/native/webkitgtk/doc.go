// Package webkitgtk loads WebKitGTK without cgo.
//
// WebKitGTK 6 with GTK 4 is preferred. WebKit2GTK 4.1 with GTK 3 is the
// fallback, which is the conda-forge webkit2gtk4.1 package. Nothing in
// this package listens on a socket.
package webkitgtk

//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind Glib --out glib_bind.go glib.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind GObject --out gobject_bind.go gobject.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind Gio --out gio_bind.go gio.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind GioOpt --soft --out gio_opt_bind.go gio_opt.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind GTK --out gtk_bind.go gtk.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind GTK4 --out gtk4_bind.go gtk4.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind GTK3 --out gtk3_bind.go gtk3.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind WebKit --out webkit_bind.go webkit.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind WebKitOpt --soft --out webkit_opt_bind.go webkit_opt.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind JSC --out jsc_bind.go jsc.h
//go:generate go run ../../../../cmd/lewkit generate header --package webkitgtk --bind Soup --soft --out soup_bind.go soup.h
