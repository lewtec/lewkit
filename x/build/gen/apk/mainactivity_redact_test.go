package apk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Splash error paths must not render the authenticated READY URL (token in query).
func TestCreate_MainActivityRedactsErrorURLs(t *testing.T) {
	out := t.TempDir()
	if err := Create(Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
	}); err != nil {
		t.Fatal(err)
	}

	page, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/PageActivity.java"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(page)

	if !strings.Contains(s, "String redactUrlForDisplay") {
		t.Fatal("PageActivity missing redactUrlForDisplay helper")
	}
	if !strings.Contains(s, "redactUrlForDisplay(raw)") {
		t.Fatal("expected redactUrlForDisplay on error paths")
	}
	// Raw request.url must not appear in splash detail strings.
	if strings.Contains(s, "\n${request.url}") {
		t.Fatal("load error path still interpolates raw request.url")
	}
	if strings.Contains(s, "append(reqUrl)") {
		// reqUrl must be assigned from redactUrlForDisplay
		if !strings.Contains(s, "val reqUrl = redactUrlForDisplay(") {
			t.Fatal("HTTP error reqUrl is not redacted")
		}
	}
	// Generated source should not hardcode a sample token query in error UI.
	if strings.Contains(s, "?token=") {
		t.Fatal("MainActivity source embeds ?token=")
	}
}
