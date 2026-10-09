package linux

import (
	"strings"
	"testing"
)

func TestDesktopFile(t *testing.T) {
	body := DesktopFile("Demo App", "br.tec.lew.demo", "1.2.3", "Demo App", "icon")
	for _, line := range []string{
		"Type=Application",
		"Name=Demo App",
		`Exec="Demo App"`,
		"Icon=icon",
		"StartupWMClass=br.tec.lew.demo",
		"Terminal=false",
		"X-Lewkit-Version=1.2.3",
	} {
		if !strings.Contains(body, line) {
			t.Fatalf("missing %q in\n%s", line, body)
		}
	}
}
