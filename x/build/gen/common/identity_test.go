package common

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
)

func TestApplyHostDefaults_FillsAppNameAndGoMain(t *testing.T) {
	t.Parallel()
	got, err := ApplyHostDefaults(t.Context(), HostConfig{
		PackageID:   " br.tec.lew.counter ",
		VersionName: "1.0.0",
		VersionCode: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.PackageID != "br.tec.lew.counter" {
		t.Fatalf("PackageID = %q", got.PackageID)
	}
	if got.AppName != "counter" {
		t.Fatalf("AppName = %q", got.AppName)
	}
	if got.GoMain != "." {
		t.Fatalf("GoMain = %q", got.GoMain)
	}
	if got.VersionName != "1.0.0" || got.VersionCode != 1 {
		t.Fatalf("version = %q / %d", got.VersionName, got.VersionCode)
	}
}

func TestApplyHostDefaults_RejectsBadPackageID(t *testing.T) {
	t.Parallel()
	_, err := ApplyHostDefaults(t.Context(), HostConfig{})
	if !errors.Is(err, release.ErrAppIDRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestEncodeHostJSON(t *testing.T) {
	t.Parallel()
	raw, err := EncodeHostJSON(HostConfig{
		PackageID:   "br.tec.lew.counter",
		AppName:     "Counter",
		VersionName: "1.2.3",
		VersionCode: 4,
		GoMain:      ".",
		Icon:        "icon.png",
	}, "eletrocromo-ios")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatal("missing trailing newline")
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %v", doc["schema_version"])
	}
	if doc["generator"] != "eletrocromo-ios" {
		t.Fatalf("generator = %v", doc["generator"])
	}
	if doc["package_id"] != "br.tec.lew.counter" {
		t.Fatalf("package_id = %v", doc["package_id"])
	}
	if doc["icon"] != "icon.png" {
		t.Fatalf("icon = %v", doc["icon"])
	}
}
