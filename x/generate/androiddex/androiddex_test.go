package androiddex

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestRunEmptyDir(t *testing.T) {
	t.Parallel()
	err := Run(t.Context(), "")
	require.ErrorIs(t, err, generate.ErrDirRequired)
}

func TestRunCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := Run(ctx, t.TempDir())
	require.ErrorIs(t, err, context.Canceled)
}

func TestRunNotPackage(t *testing.T) {
	t.Parallel()
	err := Run(t.Context(), t.TempDir())
	require.ErrorIs(t, err, ErrPackage)
}

func TestRepoResourceClass(t *testing.T) {
	t.Parallel()
	ids, err := resourceIDs(t.Context(), filepath.Join(repoAPKDir(t), residsName))
	require.NoError(t, err)
	src, err := lewkitResourceJava(ids)
	require.NoError(t, err)
	require.Contains(t, src, "package lewkit;")
	require.Contains(t, src, "public static final int splash = 0x7f010000;")
	require.Contains(t, src, "public static final int splash_status = 0x7f010005;")
	require.Contains(t, src, "public static final int activity_main = 0x7f020000;")
	require.Contains(t, src, "public static final int ic_launcher = 0x7f030000;")
	require.Contains(t, src, "public static final int app_name = 0x7f040000;")
	require.Contains(t, src, "public static final int webview_http_error = 0x7f040005;")
	require.Contains(t, src, "public static final int Theme_Eletrocromo = 0x7f050000;")
	require.Contains(t, src, "public static final int file_paths = 0x7f060000;")
	require.Contains(t, src, "public static final int network_security_config = 0x7f060001;")
}

func TestResourceIDMustMatch(t *testing.T) {
	t.Parallel()
	ids := map[string]uint32{}
	for name := range resourceFields {
		ids[name] = 1
	}
	_, err := lewkitResourceJava(ids)
	require.NoError(t, err)

	ids["resIDExtra"] = 2
	_, err = lewkitResourceJava(ids)
	require.ErrorIs(t, err, ErrResource)

	delete(ids, "resIDExtra")
	delete(ids, "resIDSplash")
	_, err = lewkitResourceJava(ids)
	require.ErrorIs(t, err, ErrResource)
}

func TestRenderHostTemplate(t *testing.T) {
	t.Parallel()
	got, err := renderHostTemplate("package {{.PackageID}};\n\nimport lewkit.Host;\n\nclass MainActivity {}\n")
	require.NoError(t, err)
	require.Equal(t, "package lewkit;\n\n\nclass MainActivity {}\n", got)
	require.NotContains(t, got, "import lewkit.")
	require.NotContains(t, got, "{{.PackageID}}")

	_, err = renderHostTemplate("package lewkit;\n")
	require.ErrorIs(t, err, ErrPackage)
}

func TestResourceText(t *testing.T) {
	t.Parallel()
	src, err := resourceJavaFromText("androidx.core", ""+
		"int id accessibility_custom_action_0 0x7f0b0002\n"+
		"int[] styleable ColorStateListItem { 0x7f040001, 0x101031f }\n"+
		"int styleable ColorStateListItem_alpha 0\n")
	require.NoError(t, err)
	require.Contains(t, src, "package androidx.core;")
	require.Contains(t, src, "public static final int accessibility_custom_action_0 = 0x7f0b0002;")
	require.Contains(t, src, "public static final int[] ColorStateListItem = { 0x7f040001, 0x101031f };")
	require.Contains(t, src, "public static final int ColorStateListItem_alpha = 0;")

	_, err = resourceJavaFromText("androidx.core", "int id class 1\n")
	require.ErrorIs(t, err, ErrResource)
	_, err = resourceJavaFromText("androidx.core", "nope\n")
	require.ErrorIs(t, err, ErrResource)
}

func TestHostSources(t *testing.T) {
	t.Parallel()
	sources, err := hostSources(t.Context(), repoAPKDir(t))
	require.NoError(t, err)
	main, ok := sources["lewkit/MainActivity.java"]
	require.True(t, ok)
	require.Contains(t, main, "package lewkit;")
	require.NotContains(t, main, "{{.PackageID}}")
	require.NotContains(t, main, "import lewkit.")
	require.Contains(t, main, "R.layout.activity_main")
	host := sources["lewkit/HostActivity.java"]
	require.Contains(t, host, "requestWindowFeature(Window.FEATURE_NO_TITLE)")
	require.Contains(t, sources["lewkit/R.java"], "splash = 0x7f010000")
}

func TestCommittedDex(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join(repoAPKDir(t), dexName))
	require.NoError(t, err)
	require.NoError(t, checkDex(raw))
}

func TestCheckDexRejects(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, checkDex([]byte("dex\n038")), ErrCompile)
	raw := append([]byte("dex\n039"), make([]byte, 120)...)
	require.ErrorIs(t, checkDex(raw), ErrCompile)
}

func TestJDKVersion(t *testing.T) {
	t.Parallel()
	require.False(t, jdkVersionOK(`openjdk version "17.0.2" 2022-01-18`))
	require.True(t, jdkVersionOK(`openjdk version "17.0.14" 2025-01-21`))
	require.False(t, jdkVersionOK(`openjdk version "11.0.2"`))
	require.True(t, jdkVersionOK(`openjdk version "21.0.2"`))
	require.False(t, jdkVersionOK("no version"))
}

func TestVersionNewer(t *testing.T) {
	t.Parallel()
	require.True(t, versionNewer("34.0.0", "30.0.3"))
	require.True(t, versionNewer("android-35", "android-28"))
	require.False(t, versionNewer("34.0.0", "34.0.0"))
	require.False(t, versionNewer("30.0.3", "34.0.0"))
}

func TestRebuild(t *testing.T) {
	if os.Getenv("LEWKIT_ANDROIDDEX") == "" {
		test.Skip(t, "LEWKIT_ANDROIDDEX")
	}
	src := repoAPKDir(t)
	dir := t.TempDir()
	require.NoError(t, copyFile(filepath.Join(src, residsName), filepath.Join(dir, residsName)))
	require.NoError(t, copyTree(filepath.Join(src, "template"), filepath.Join(dir, "template")))
	require.NoError(t, Run(t.Context(), dir))
	raw, err := os.ReadFile(filepath.Join(dir, dexName))
	require.NoError(t, err)
	require.NoError(t, checkDex(raw))
	require.Contains(t, string(raw), "requestWindowFeature")
	_, err = os.Stat(filepath.Join(repoAPKDir(t), dexName))
	require.NoError(t, err)
}

func repoAPKDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "build", "gen", "apk")
	_, err := os.Stat(filepath.Join(dir, residsName))
	require.NoError(t, err)
	return dir
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}
