package linux

import (
	"fmt"
	"strings"
)

// DesktopFile is the FreeDesktop entry for one bundled executable.
// execPath and iconPath are absolute paths inside the app directory.
func DesktopFile(name, id, version, execPath, iconPath string) string {
	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Version=1.0\n")
	fmt.Fprintf(&b, "Name=%s\n", desktopValue(name))
	fmt.Fprintf(&b, "Exec=%s\n", quoteExec(execPath))
	fmt.Fprintf(&b, "Icon=%s\n", desktopValue(iconPath))
	fmt.Fprintf(&b, "StartupWMClass=%s\n", desktopValue(id))
	b.WriteString("Terminal=false\n")
	b.WriteString("Categories=Utility;\n")
	if strings.TrimSpace(version) != "" {
		fmt.Fprintf(&b, "X-Lewkit-Version=%s\n", desktopValue(version))
	}
	return b.String()
}

func desktopValue(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`).Replace(s)
}

func quoteExec(path string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range path {
		switch r {
		case '\\', '"', '`', '$':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

func appRunScript(product string) string {
	return "#!/bin/sh\n" +
		"dir=$(CDPATH= cd -- \"$(dirname \"$0\")\" && pwd) || exit 1\n" +
		"exec \"$dir/" + product + "\" \"$@\"\n"
}
