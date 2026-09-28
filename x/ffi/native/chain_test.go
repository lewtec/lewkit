package native

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchDirsPutsEnvFirst(t *testing.T) {
	t.Setenv("LEWKIT_LIB", "/tmp/lewkit-a"+string(filepath.ListSeparator)+"/tmp/lewkit-b")
	t.Setenv("LD_LIBRARY_PATH", "/tmp/lewkit-b"+string(filepath.ListSeparator)+"/tmp/ld")
	dirs := SearchDirs("/tmp/extra")
	require.Equal(t, "/tmp/extra", dirs[0])
	require.Equal(t, "/tmp/lewkit-a", dirs[1])
	require.Equal(t, "/tmp/lewkit-b", dirs[2])
	if runtime.GOOS != "windows" {
		require.Contains(t, dirs, "/tmp/ld")
		require.Contains(t, dirs, "/run/current-system/sw/lib")
	}
	require.Equal(t, len(dirs), len(unique(dirs)))
}

func unique(dirs []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, dir := range dirs {
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}

func TestOpenChainTriesBareNameThenDirsAndRemembers(t *testing.T) {
	t.Setenv("LEWKIT_LIB", "/tmp/lewkit-chain")
	var paths []string
	swapLoader(t, func(path string, _ int) (uintptr, error) {
		paths = append(paths, path)
		if path == filepath.Join("/tmp/lewkit-chain", "libgood.so") {
			return 4, nil
		}
		return 0, errMissing
	})
	first, err := OpenChain(Lazy, "libgood.so")
	require.NoError(t, err)
	require.Equal(t, uintptr(4), first)
	require.Equal(t, "libgood.so", paths[0])
	require.Contains(t, paths, filepath.Join("/tmp/lewkit-chain", "libgood.so"))
	n := len(paths)
	second, err := OpenChain(Lazy, "libgood.so")
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, paths, n)
}

func TestOpenChainKeepsTheFailure(t *testing.T) {
	var n int
	swapLoader(t, func(string, int) (uintptr, error) {
		n++
		return 0, errMissing
	})
	_, err := OpenChain(Lazy, "libmissing-once.so")
	require.ErrorIs(t, err, errMissing)
	calls := n
	require.NotZero(t, calls)
	_, err = OpenChain(Lazy, "libmissing-once.so")
	require.ErrorIs(t, err, errMissing)
	require.Equal(t, calls, n)
}

func TestOpenChainLeavesPathsAlone(t *testing.T) {
	var paths []string
	swapLoader(t, func(path string, _ int) (uintptr, error) {
		paths = append(paths, path)
		return 0, errMissing
	})
	path := filepath.Join(t.TempDir(), "libonly.so")
	_, err := OpenChain(Lazy, path)
	require.ErrorIs(t, err, errMissing)
	require.Equal(t, []string{path}, paths)
}

func TestOpenInTriesExtraDirsFirst(t *testing.T) {
	var paths []string
	swapLoader(t, func(path string, _ int) (uintptr, error) {
		paths = append(paths, path)
		if path == filepath.Join("/tmp/extra-chain", "libextra.so") {
			return 5, nil
		}
		return 0, errMissing
	})
	handle, err := OpenIn(Lazy, []string{"/tmp/extra-chain"}, "libextra.so")
	require.NoError(t, err)
	require.Equal(t, uintptr(5), handle)
	require.Equal(t, "libextra.so", paths[0])
	require.Equal(t, filepath.Join("/tmp/extra-chain", "libextra.so"), paths[1])
}

func TestProcOfRemembersTheLookup(t *testing.T) {
	var n int
	swapLoader(t, func(string, int) (uintptr, error) {
		n++
		return 0, errMissing
	})
	proc := ProcOf("liblewkit-missing.so", "sym")
	require.ErrorIs(t, proc.Find(), errMissing)
	calls := n
	require.ErrorIs(t, proc.Find(), errMissing)
	require.Equal(t, calls, n)
}

func TestSingletonKeepsTheValue(t *testing.T) {
	var n int
	load := Singleton(func() (int, error) {
		n++
		return 9, errMissing
	})
	value, err := load()
	require.Equal(t, 9, value)
	require.ErrorIs(t, err, errMissing)
	value, err = load()
	require.Equal(t, 9, value)
	require.ErrorIs(t, err, errMissing)
	require.Equal(t, 1, n)
}
