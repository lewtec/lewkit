package linux

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

func TestHostConfigRoundTrip(t *testing.T) {
	cfg := Config{
		PackageID:   "br.tec.lew.counter",
		AppName:     "Counter",
		VersionName: "1.2.3",
		VersionCode: 4,
		GoMain:      ".",
		Icon:        "icon.png",
	}
	require.Equal(t, cfg, configFromHost(cfg.hostConfig()))
	require.Equal(t, common.HostConfig{
		PackageID:   cfg.PackageID,
		AppName:     cfg.AppName,
		VersionName: cfg.VersionName,
		VersionCode: cfg.VersionCode,
		GoMain:      cfg.GoMain,
		Icon:        cfg.Icon,
	}, cfg.hostConfig())
}
