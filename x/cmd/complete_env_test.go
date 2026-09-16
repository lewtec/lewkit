package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWordsAfterCommand(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{line: "", want: []string{""}},
		{line: "tool", want: nil},
		{line: "tool ", want: []string{""}},
		{line: "tool add", want: []string{"add"}},
		{line: "tool add ", want: []string{"add", ""}},
		{line: "tool add --na", want: []string{"add", "--na"}},
		{line: "  tool   add   --na", want: []string{"add", "--na"}},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			assert.Equal(t, tc.want, wordsAfterCommand(tc.line))
		})
	}
}

func TestWordsFromCompletionEnv(t *testing.T) {
	env := map[string]string{
		"COMP_LINE":  "tool add --na",
		"COMP_POINT": "8",
	}
	getenv := func(key string) string { return env[key] }
	words, ok := wordsFromCompletionEnv(getenv)
	require.True(t, ok)
	assert.Equal(t, []string{"add"}, words)
}

func TestWordsFromCompletionEnvAbsent(t *testing.T) {
	_, ok := wordsFromCompletionEnv(func(string) string { return "" })
	assert.False(t, ok)
}

func TestWriteEnvCompletions(t *testing.T) {
	env := map[string]string{"COMP_LINE": "tool add --na"}
	var buf bytes.Buffer
	ok := writeEnvCompletions[appArgs](&buf, func(key string) string { return env[key] })
	require.True(t, ok)
	assert.Equal(t, "--name\n", buf.String())
}

func TestWriteEnvCompletionsAbsent(t *testing.T) {
	var buf bytes.Buffer
	ok := writeEnvCompletions[appArgs](&buf, func(string) string { return "" })
	assert.False(t, ok)
	assert.Empty(t, buf.String())
}

func TestBashCompleteLine(t *testing.T) {
	got := BashCompleteLine("lewkit")
	assert.Equal(t, "complete -C 'lewkit' 'lewkit'", got)
	assert.True(t, strings.HasPrefix(BashCompleteLine("lewkit's"), "complete -C "))
}
