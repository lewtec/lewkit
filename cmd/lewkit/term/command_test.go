package term

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTermUsage(t *testing.T) {
	text, err := cmd.Usage[Cmd]("lewkit term")
	require.NoError(t, err)
	assert.Contains(t, text, "serve")
	assert.Contains(t, text, "ubersession")
	assert.Contains(t, text, "attach")
	assert.Contains(t, text, "rpc")
}

func TestTermParse(t *testing.T) {
	serve := cmd.ParseOK[Cmd](t, "serve", "--socket", "/tmp/term.sock")
	require.NotNil(t, serve.serve)
	assert.Equal(t, "/tmp/term.sock", serve.serve.socket.Value())

	attach := cmd.ParseOK[Cmd](t, "attach", "--pane", "pane-1")
	require.NotNil(t, attach.attach)
	assert.Equal(t, "pane-1", attach.attach.pane.Value())

	call := cmd.ParseOK[Cmd](t, "rpc", "pane.list", `{"ok":true}`)
	require.NotNil(t, call.rpc)
	assert.Equal(t, "pane.list", call.rpc.method.Value())
	require.NotNil(t, call.rpc.params)
	assert.Equal(t, `{"ok":true}`, call.rpc.params.Value())
}
