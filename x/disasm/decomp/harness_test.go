package decomp

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHarnessMatchUsesSubscriptionProcess(t *testing.T) {
	artifact := openELF(t, []byte{0xc3})
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	body := echoCandidate(function, "sub_1000", "void sub_1000(void)")
	wrapped, err := json.Marshal(map[string]string{
		"text":       body,
		"stopReason": "end_turn",
	})
	require.NoError(t, err)
	artifact.Attach(Harness{
		Model: "grok-4.6",
		Dir:   t.TempDir(),
		Run: func(_ context.Context, call HarnessCall) ([]byte, error) {
			assert.Equal(t, "grok", call.Path)
			assert.Equal(t, []string{"--model", "grok-4.6"}, call.Args[:2])
			assert.Contains(t, call.Args, "--sandbox")
			assert.Contains(t, call.Args, "read-only")
			assert.Contains(t, call.Args, "--permission-mode")
			assert.Contains(t, call.Args, "dontAsk")
			assert.Contains(t, call.Args, "--verbatim")
			assert.Contains(t, call.Args, "--output-format")
			assert.Contains(t, call.Args, "json")
			assert.Contains(t, call.Args, "--json-schema")
			assert.Contains(t, call.Args, "--max-turns")
			assert.Contains(t, call.Args, "1")
			assert.Contains(t, call.Args, "--system-prompt-override")
			assert.NotContains(t, call.Args, "auth.json")
			promptPath := flagValue(t, call.Args, "--prompt-file")
			prompt, err := os.ReadFile(promptPath)
			require.NoError(t, err)
			assert.Contains(t, string(prompt), "Match this function.")
			return wrapped, nil
		},
	})
	result, err := artifact.Match(t.Context(), 0x1000, MatchConfig{Rounds: 1})
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.Equal(t, "sub_1000", result.Name)
}

func TestHarnessReportsSignedInFailure(t *testing.T) {
	harness := Harness{
		Dir: t.TempDir(),
		Run: func(context.Context, HarnessCall) ([]byte, error) {
			return []byte(`{"type":"error","message":"not signed in"}`), errors.New("exit status 1")
		},
	}
	_, err := harness.Complete(t.Context(), Prompt{System: "reply in json", User: "match"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not signed in")
}

func TestHarnessAcceptsBareCandidate(t *testing.T) {
	const candidate = `{"name":"sub_1000","signature":"void sub_1000(void)","source":"void sub_1000(void) { return; }","blocks":[]}`
	harness := Harness{
		Dir: t.TempDir(),
		Run: func(context.Context, HarnessCall) ([]byte, error) {
			return []byte("note\n" + candidate + "\n"), nil
		},
	}
	text, err := harness.Complete(t.Context(), Prompt{User: "match"})
	require.NoError(t, err)
	assert.JSONEq(t, candidate, text)
}

func TestHarnessMissingExecutable(t *testing.T) {
	_, err := (Harness{Path: filepath.Join(t.TempDir(), "grok")}).Complete(t.Context(), Prompt{User: "match"})
	require.Error(t, err)
	assert.ErrorIs(t, err, execdriver.ErrNotFound)
}

func flagValue(t *testing.T, args []string, name string) string {
	t.Helper()
	for index, arg := range args {
		if arg == name {
			require.Less(t, index+1, len(args))
			return args[index+1]
		}
	}
	t.Fatalf("missing %s", name)
	return ""
}
