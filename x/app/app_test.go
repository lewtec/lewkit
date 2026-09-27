package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
