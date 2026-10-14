package brightnessctl

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/stretchr/testify/require"
)

func TestParseStatus(t *testing.T) {
	dev, err := parseStatus("intel_backlight,backlight,50,25%,200\n")
	require.NoError(t, err)
	if dev.Name != "intel_backlight" || dev.Brightness != 0.25 {
		require.FailNowf(t, "", "%+v", dev)
	}
	_, err = parseStatus("short\nbad,class,1,nope%,2\n")
	require.ErrorIs(t, err, brightness.ErrDeviceNotFound)
}
