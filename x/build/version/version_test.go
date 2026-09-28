package version

import (
	"strings"
	"testing"
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
			t.Errorf("%q → %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestAndroidCodeFrom_Semver(t *testing.T) {
	if c := AndroidCodeFrom("v1.2.3", 0); c != 1_002_003 {
		t.Fatalf("got %d", c)
	}
	if c := AndroidCodeFrom("0.1.0", 99); c != 1000 {
		// 0*1e6 + 1*1e3 + 0 = 1000
		t.Fatalf("got %d", c)
	}
	if c := AndroidCodeFrom("devel", 42); c != 42 {
		t.Fatalf("git count fallback: %d", c)
	}
	if c := AndroidCodeFrom("abcdef", 0); c != 1 {
		t.Fatalf("default: %d", c)
	}
}

func TestResolve_Defaults(t *testing.T) {
	// Do not mutate package vars permanently beyond test — save/restore.
	oldV, oldC, oldD, oldB := Version, Commit, Date, BuiltBy
	t.Cleanup(func() {
		Version, Commit, Date, BuiltBy = oldV, oldC, oldD, oldB
	})
	Version, Commit, Date, BuiltBy = "devel", "", "", ""
	info := Resolve()
	if info.Version == "" {
		t.Fatal("empty version")
	}
	s := info.String()
	if !strings.Contains(s, info.Version) {
		t.Fatalf("string: %s", s)
	}
}

func TestGoBuildLdflags(t *testing.T) {
	lf := Info{Version: "v1.0.0", Commit: "abc", Date: "2026-01-01T00:00:00Z", BuiltBy: "test"}.GoBuildLdflags()
	if !strings.Contains(lf, "-s -w") {
		t.Fatal(lf)
	}
	if !strings.Contains(lf, "Version=v1.0.0") {
		t.Fatal(lf)
	}
	if !strings.Contains(lf, "BuiltBy=test") {
		t.Fatal(lf)
	}
}
