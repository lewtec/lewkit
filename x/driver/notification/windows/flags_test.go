package windows

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestInfoFlags(t *testing.T) {
	if infoFlags("critical") != niifError {
		require.FailNowf(t, "", "critical")
	}
	if infoFlags("low") != niifInfo|niifNoSound {
		require.FailNowf(t, "", "low")
	}
	if infoFlags("normal") != niifInfo || infoFlags("") != niifInfo {
		require.FailNowf(t, "", "normal")
	}
}

func TestNotifyIconLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip()
	}
	var data notifyIconData
	if unsafe.Sizeof(data) != 952 {
		require.FailNowf(t, "", "size %d", unsafe.Sizeof(data))
	}
	if unsafe.Offsetof(data.infoFlags) != 948 {
		require.FailNowf(t, "", "infoFlags %d", unsafe.Offsetof(data.infoFlags))
	}
}
