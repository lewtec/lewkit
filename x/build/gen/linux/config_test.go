package linux

import (
	"testing"

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
	if configFromHost(cfg.hostConfig()) != cfg {
		t.Fatalf("round trip = %#v", configFromHost(cfg.hostConfig()))
	}
	if cfg.hostConfig() != (common.HostConfig{
		PackageID:   cfg.PackageID,
		AppName:     cfg.AppName,
		VersionName: cfg.VersionName,
		VersionCode: cfg.VersionCode,
		GoMain:      cfg.GoMain,
		Icon:        cfg.Icon,
	}) {
		t.Fatal("host fields differ")
	}
}
