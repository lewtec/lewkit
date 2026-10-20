package lewkit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCgoExportArg(t *testing.T) {
	got, ok := cgoExportArg("'/tmp/wrap' '/tmp/_cgo_android_export.go'")
	if !ok || got != "/tmp/_cgo_android_export.go" {
		require.FailNowf(t, "", "%q %v", got, ok)
	}
	got, ok = cgoExportArg("'/tmp/wrap' '/tmp/_cgo_clib_export.go'")
	if !ok || got != "/tmp/_cgo_clib_export.go" {
		require.FailNowf(t, "", "clib %q %v", got, ok)
	}
	if _, ok := cgoExportArg("'/tmp/wrap' '/tmp/_cgo_other.go'"); ok {
		require.FailNow(t, fmt.Sprint("accepted _cgo_other.go"))
	}
	if _, ok := cgoExportArg("wrap"); ok {
		require.FailNow(t, fmt.Sprint("accepted wrap"))
	}
}
