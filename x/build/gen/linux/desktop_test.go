package linux

import (
	"strings"
	"testing"
)

func TestDesktopFile(t *testing.T) {
	body := DesktopFile("Demo App", "br.tec.lew.demo", "1.2.3", "/opt/Demo App/Demo", "/opt/Demo App/icon.png")
	for _, line := range []string{
		"Type=Application",
		"Name=Demo App",
		`Exec="/opt/Demo App/Demo"`,
		"Icon=/opt/Demo App/icon.png",
		"StartupWMClass=br.tec.lew.demo",
		"Terminal=false",
		"X-Lewkit-Version=1.2.3",
	} {
		if !strings.Contains(body, line) {
			t.Fatalf("missing %q in\n%s", line, body)
		}
	}
}

func TestAppRunNamesProduct(t *testing.T) {
	script := appRunScript("Demo")
	if !strings.Contains(script, `exec "$dir/Demo"`) {
		t.Fatalf("script %s", script)
	}
}
