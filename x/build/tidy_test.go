package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

// TestConsumerToolchainTidy imports the copied linker from a module outside
// this one. x/build reaches that linker through x/toolchain/go, and go mod
// tidy reads outbuf_bsd.go on every host. The freebsd and netbsd lists must
// resolve std/syscall/unix inside this module.
func TestConsumerToolchainTidy(t *testing.T) {
	goBin := test.Need(t, "go")
	root := findModuleRoot(t)
	mod := modulePath(t, root)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(consumerGoMod(t, mod, root)), 0o644))
	src := "package consumer\n\nimport _ \"" + mod + "/x/toolchain/go/src/link/ld\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "consumer.go"), []byte(src), 0o644))

	runGo(t, goBin, dir, nil, "mod", "tidy")
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	require.NotContains(t, string(body), mod+"/")

	link := mod + "/x/toolchain/go/src/link/ld"
	unix := mod + "/x/toolchain/go/src/std/syscall/unix"
	for _, target := range linkerTargets() {
		listed := runGo(t, goBin, dir, target.env(), "list", "-deps", link)
		require.Contains(t, string(listed), unix)
		archive := filepath.Join(t.TempDir(), "ld.a")
		runGo(t, goBin, root, target.env(), "build", "-o", archive, link)
	}
	for _, target := range []goTarget{
		{GOOSLinux, GOARCHAMD64},
		{GOOSDarwin, GOARCHAMD64},
		{GOOSWindows, GOARCHAMD64},
	} {
		archive := filepath.Join(t.TempDir(), "unix.a")
		runGo(t, goBin, root, target.env(), "build", "-o", archive, unix)
	}
}

type goTarget struct {
	GOOS   GOOS
	GOARCH GOARCH
}

func linkerTargets() []goTarget {
	return []goTarget{
		{GOOSFreebsd, GOARCHAMD64},
		{GOOSNetbsd, GOARCHARM64},
	}
}

func (g goTarget) env() map[string]string {
	return map[string]string{
		"GOOS":   g.GOOS.String(),
		"GOARCH": g.GOARCH.String(),
	}
}

func consumerGoMod(t *testing.T, mod, root string) string {
	t.Helper()
	return strings.Join([]string{
		"module example.com/lewkitconsumer",
		"",
		"go " + goVersion(t, root),
		"",
		"require " + mod + " v0.0.0",
		"",
		"replace " + mod + " => " + filepath.ToSlash(root),
		"",
	}, "\n")
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent)
		dir = parent
	}
}

func modulePath(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	require.FailNow(t, "go.mod has no module path")
	return ""
}

func goVersion(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "go" {
			return fields[1]
		}
	}
	require.FailNow(t, "go.mod has no go version")
	return ""
}

func runGo(t *testing.T, goBin, dir string, overlay map[string]string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), goBin, args...)
	cmd.Dir = dir
	cmd.Env = goEnv(overlay)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "go %s:\n%s", strings.Join(args, " "), out)
	return out
}

func goEnv(overlay map[string]string) []string {
	if overlay == nil {
		overlay = map[string]string{}
	}
	overlay["CGO_ENABLED"] = "0"
	overlay["GO111MODULE"] = "on"
	skip := map[string]bool{}
	for key := range overlay {
		skip[key] = true
	}
	env := make([]string, 0, len(os.Environ())+len(overlay))
	for _, kv := range os.Environ() {
		key, _, ok := strings.Cut(kv, "=")
		if !ok || skip[key] {
			continue
		}
		env = append(env, kv)
	}
	keys := make([]string, 0, len(overlay))
	for key := range overlay {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+overlay[key])
	}
	return env
}
