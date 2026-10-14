package version

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAndroidName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"devel", "0.0.0-devel"},
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v1.2.3-5-gabcdef", "1.2.3-5-gabcdef"},
		{"", "0.0.0-devel"},
	}
	for _, tc := range cases {
		got := Info{Version: tc.in}.AndroidName()
		if got != tc.want {
			assert.Failf(t, "", "%q → %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestAndroidCodeFrom_Semver(t *testing.T) {
	if c := AndroidCodeFrom("v1.2.3", 0); c != 1_002_003 {
		require.FailNowf(t, "", "got %d", c)
	}
	if c := AndroidCodeFrom("0.1.0", 99); c != 1000 {
		// 0*1e6 + 1*1e3 + 0 = 1000
		require.FailNowf(t, "", "got %d", c)
	}
	if c := AndroidCodeFrom("devel", 42); c != 42 {
		require.FailNowf(t, "", "git count fallback: %d", c)
	}
	if c := AndroidCodeFrom("abcdef", 0); c != 1 {
		require.FailNowf(t, "", "default: %d", c)
	}
}

func TestResolve_Defaults(t *testing.T) {
	oldC, oldD, oldB := Commit, Date, BuiltBy
	t.Cleanup(func() {
		Commit, Date, BuiltBy = oldC, oldD, oldB
	})
	Commit, Date, BuiltBy = "", "", ""
	info := Resolve(t.Context())
	if info.Version == "" {
		require.FailNow(t, fmt.Sprint("empty version"))
	}
	s := info.String()
	if !strings.Contains(s, info.Version) {
		require.FailNowf(t, "", "string: %s", s)
	}
}

func TestGoBuildLdflags(t *testing.T) {
	lf := Info{Version: "v1.0.0", Commit: "abc", Date: "2026-01-01T00:00:00Z", BuiltBy: "test"}.GoBuildLdflags()
	if !strings.Contains(lf, "-s -w") {
		require.FailNow(t, fmt.Sprint(lf))
	}
	if !strings.Contains(lf, "github.com/lewtec/lewkit/x/release.version=v1.0.0") {
		require.FailNow(t, fmt.Sprint(lf))
	}
	if strings.Contains(lf, "x/build/version.Version=") {
		require.FailNow(t, fmt.Sprint(lf))
	}
	if !strings.Contains(lf, "BuiltBy=test") {
		require.FailNow(t, fmt.Sprint(lf))
	}
	withID := Info{Version: "v1.0.0", Commit: "abc"}.WithAppID("br.tec.lew.app")
	if !strings.Contains(withID, "x/release.version=v1.0.0") {
		require.FailNow(t, fmt.Sprint(withID))
	}
	if !strings.Contains(withID, "x/release.appID=br.tec.lew.app") {
		require.FailNow(t, fmt.Sprint(withID))
	}
	if strings.Count(withID, "x/release.version=") != 1 {
		require.FailNow(t, fmt.Sprint(withID))
	}
}
