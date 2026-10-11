package apk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Splash error paths must not render the authenticated READY URL (token in query).
func TestCreate_MainActivityRedactsErrorURLs(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, Create(t.Context(), Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
	}))

	page, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/PageActivity.java"))
	require.NoError(t, err)
	s := string(page)

	require.Contains(t, s, "String redactUrlForDisplay")
	require.Contains(t, s, "redactUrlForDisplay(raw)")
	// Raw request.url must not appear in splash detail strings.
	require.NotContains(t, s, "\n${request.url}")
	if strings.Contains(s, "append(reqUrl)") {
		// reqUrl must be assigned from redactUrlForDisplay
		require.Contains(t, s, "val reqUrl = redactUrlForDisplay(")
	}
	// Generated source should not hardcode a sample token query in error UI.
	require.NotContains(t, s, "?token=")
}
