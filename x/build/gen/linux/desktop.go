package linux

import (
	"fmt"
	"strings"
)

// DesktopFile is the FreeDesktop entry carried inside one AppImage.
// execName and iconName are names in that file, the way an AppImage names them.
func DesktopFile(name, id, version, execName, iconName string) string {
	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	b.WriteString("Type=Application\n")
	b.WriteString("Version=1.0\n")
	fmt.Fprintf(&b, "Name=%s\n", desktopValue(name))
	fmt.Fprintf(&b, "Exec=%s\n", execField(execName))
	fmt.Fprintf(&b, "Icon=%s\n", desktopValue(iconName))
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

func execField(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\"'\\$`") {
		return quoteExec(s)
	}
	return s
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
