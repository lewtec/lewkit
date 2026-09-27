package exec_test

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/stretchr/testify/require"
)

func TestMustRunCapturesStdout(t *testing.T) {
	cmd := execdriver.MustRun(t.Context(), "sh", "-c", "echo hi")
	require.Nil(t, cmd.Stdout)
	require.NotNil(t, cmd.Stderr)
	out, err := cmd.Output()
	require.NoError(t, err)
	require.Equal(t, "hi\n", string(out))
}

func TestMustRunStderrReachesSession(t *testing.T) {
	session, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	var lines []string
	session.SetLinePrint(func(line string) { lines = append(lines, line) })
	cmd := execdriver.MustRun(ctx, "sh", "-c", "echo hello >&2")
	require.NoError(t, cmd.Run())
	require.Equal(t, []string{"hello"}, lines)
}

func TestWhichAndRequireBinary(t *testing.T) {
	path, err := execdriver.Which(t.Context(), "sh")
	require.NoError(t, err)
	require.NotEmpty(t, path)
	require.True(t, execdriver.IsBinaryAvailable(t.Context(), "sh"))

	err = execdriver.RequireBinary(t.Context(), "lewkit-missing-exec-binary")
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestReplacedStderrIsNotALiveRow(t *testing.T) {
	session, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	session.SetLinePrint(func(string) {})
	cmd := execdriver.MustRun(ctx, "sh", "-c", "echo kept; echo dropped >&2")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	require.NoError(t, err)
	require.Equal(t, "kept\n", string(out))
	require.Equal(t, "dropped\n", string(stderr.String()))
	require.Empty(t, session.LiveLines())
}

func TestProductionSpawnsUseTheDriver(t *testing.T) {
	root := moduleRoot(t)
	var offenders []string
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\bexec\.Command(?:Context)?\(`),
		regexp.MustCompile(`\bexec\.LookPath\(`),
	}
	allowed := map[string]bool{
		filepath.Join("x", "driver", "exec", "exec.go"):               true,
		filepath.Join("x", "driver", "exec", "native", "provider.go"): true,
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if allowed[rel] {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		for _, pattern := range patterns {
			if pattern.MatchString(text) {
				offenders = append(offenders, rel)
				return nil
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, offenders)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent)
		dir = parent
	}
}
