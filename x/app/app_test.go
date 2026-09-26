package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoopbackLinkUsesIPv4(t *testing.T) {
	require.Equal(t, "http://127.0.0.1:9/?token=abc", loopbackLink("http://[::1]:9", "abc"))
}

func TestRunNoUIStopsWithContext(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "1")
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.app")
	ctx, cancel := contextWithCancel(t)
	done := make(chan error, 1)
	go func() {
		done <- App{ID: "br.tec.lew.app", NoUI: true}.Run(ctx)
	}()
	cancel()
	require.NoError(t, <-done)
}
