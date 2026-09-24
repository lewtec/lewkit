// Package portal calls an xdg-desktop-portal file chooser implementation.
//
// GTK is the gtk backend. KDE is the Qt file dialog shipped by the KDE backend.
package portal

import (
	"os"
	"strings"
)

const (
	// GTK is the gtk portal file chooser.
	GTK = "org.freedesktop.impl.portal.desktop.gtk"
	// KDE is the Qt file dialog from the KDE portal backend.
	KDE = "org.freedesktop.impl.portal.desktop.kde"

	objectPath = "/org/freedesktop/portal/desktop"
	iface      = "org.freedesktop.impl.portal.FileChooser"
	appID      = "lewkit"
)

// PreferQt reports a KDE or Plasma session, where the Qt dialog should win.
func PreferQt() bool {
	desk := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP") + ":" + os.Getenv("XDG_SESSION_DESKTOP"))
	return strings.Contains(desk, "kde") || strings.Contains(desk, "plasma")
}
