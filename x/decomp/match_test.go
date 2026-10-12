package decomp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchAcceptsEcho(t *testing.T) {
	artifact := openELF(t, []byte{0x90, 0xc3})
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	model := &scriptModel{replies: []string{echoCandidate(function, "start", "void start(void)")}}
	artifact.project.Attach(model)
	result, err := artifact.Match(t.Context(), 0x1000, MatchConfig{Rounds: 2})
	require.NoError(t, err)
	assert.True(t, result.Matched)
	assert.Equal(t, 1, result.Rounds)
	stored, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "start", stored.Name)
	assert.Contains(t, stored.Source, "void start")
}

func TestMatchRetriesThenAccepts(t *testing.T) {
	artifact := openELF(t, []byte{0x90, 0xc3})
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	wrong := `{"name":"start","signature":"void start(void)","source":"void start(void) { return; }","blocks":[]}`
	model := &scriptModel{replies: []string{wrong, echoCandidate(function, "start", "void start(void)")}}
	artifact.project.Attach(model)
	result, err := artifact.Match(t.Context(), 0x1000, MatchConfig{Rounds: 3})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Rounds)
	require.GreaterOrEqual(t, len(model.seen), 2)
	assert.Contains(t, model.seen[1].User, "block 0x1000 missing")
}

func TestMatchUnmatched(t *testing.T) {
	artifact := openELF(t, []byte{0xc3})
	artifact.project.Attach(&scriptModel{replies: []string{"no json", "still no"}})
	result, err := artifact.Match(t.Context(), 0x1000, MatchConfig{Rounds: 2})
	assert.ErrorIs(t, err, ErrUnmatched)
	assert.False(t, result.Matched)
	assert.Equal(t, 2, result.Rounds)
	assert.NotEmpty(t, result.Residuals)
}

func TestMatchRequiresModel(t *testing.T) {
	artifact := openELF(t, []byte{0xc3})
	_, err := artifact.Match(t.Context(), 0x1000, MatchConfig{})
	assert.ErrorIs(t, err, ErrNoModel)
}

func TestChatRejectsEmptyKey(t *testing.T) {
	_, err := (Chat{Endpoint: "http://127.0.0.1", Model: "grok"}).Complete(context.Background(), Prompt{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key")
}

func echoCandidate(function Function, name, signature string) string {
	source := signature + " {\n  return;\n}\n"
	blocks := ""
	for index, block := range function.Blocks {
		if index > 0 {
			blocks += ","
		}
		blocks += `{"address":"` + formatAddress(block.Address) + `","kind":"` + block.Kind + `"`
		if block.Target != 0 {
			blocks += `,"target":"` + formatAddress(block.Target) + `"`
		}
		if block.Fall != 0 {
			blocks += `,"fall":"` + formatAddress(block.Fall) + `"`
		}
		blocks += "}"
	}
	return `{"name":"` + name + `","signature":"` + signature + `","source":"` + jsonEscape(source) + `","blocks":[` + blocks + `]}`
}

func jsonEscape(text string) string {
	out := ""
	for _, char := range text {
		switch char {
		case '"':
			out += `\"`
		case '\n':
			out += `\n`
		default:
			out += string(char)
		}
	}
	return out
}

type scriptModel struct {
	replies []string
	seen    []Prompt
}

func (model *scriptModel) Complete(_ context.Context, prompt Prompt) (string, error) {
	model.seen = append(model.seen, prompt)
	if len(model.replies) == 0 {
		return "", errors.New("no reply")
	}
	reply := model.replies[0]
	model.replies = model.replies[1:]
	return reply, nil
}
