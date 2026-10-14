package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
	"github.com/stretchr/testify/require"
)

func TestApplyHostDefaults_FillsAppNameAndGoMain(t *testing.T) {
	t.Parallel()
	got, err := ApplyHostDefaults(t.Context(), HostConfig{
		PackageID:   " br.tec.lew.counter ",
		VersionName: "1.0.0",
		VersionCode: 1,
	})
	require.NoError(t, err)
	if got.PackageID != "br.tec.lew.counter" {
		require.FailNowf(t, "", "PackageID = %q", got.PackageID)
	}
	if got.AppName != "counter" {
		require.FailNowf(t, "", "AppName = %q", got.AppName)
	}
	if got.GoMain != "." {
		require.FailNowf(t, "", "GoMain = %q", got.GoMain)
	}
	if got.VersionName != "1.0.0" || got.VersionCode != 1 {
		require.FailNowf(t, "", "version = %q / %d", got.VersionName, got.VersionCode)
	}
}

func TestApplyHostDefaults_RejectsBadPackageID(t *testing.T) {
	t.Parallel()
	_, err := ApplyHostDefaults(t.Context(), HostConfig{})
	require.ErrorIs(t, err, release.ErrAppIDRequired)
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
	require.NoError(t, err)
	if !strings.HasSuffix(string(raw), "\n") {
		require.FailNow(t, fmt.Sprint("missing trailing newline"))
	}
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	if doc["schema_version"] != float64(1) {
		require.FailNowf(t, "", "schema_version = %v", doc["schema_version"])
	}
	if doc["generator"] != "eletrocromo-ios" {
		require.FailNowf(t, "", "generator = %v", doc["generator"])
	}
	if doc["package_id"] != "br.tec.lew.counter" {
		require.FailNowf(t, "", "package_id = %v", doc["package_id"])
	}
	if doc["icon"] != "icon.png" {
		require.FailNowf(t, "", "icon = %v", doc["icon"])
	}
}
