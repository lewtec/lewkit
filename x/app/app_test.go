package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnnounceReadyWritesFile(t *testing.T) {
	path := t.TempDir() + "/ready.url"
	t.Setenv("ELETROCROMO_READY_FILE", path)
	announceReady("http://127.0.0.1:9/?token=abc")
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:9/?token=abc\n", string(body))
}

func TestLoopbackLinkUsesIPv4(t *testing.T) {
	require.Equal(t, "http://127.0.0.1:9/?token=abc", loopbackLink("http://[::1]:9", "abc"))
}

func TestRunPanicsWithoutStamp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = App{NoUI: true}.Run(t.Context())
}
