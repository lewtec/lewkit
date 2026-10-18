package term

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	mux "github.com/lewtec/lewkit/x/term"
	"github.com/lewtec/lewkit/x/term/schema"
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

func TestRPCParams(t *testing.T) {
	write := cmd.ParseOK[Cmd](t, "rpc", "pane.write", `{"id":"p","data":"\u0003","rows":24,"cols":80,"extra":1}`)
	msg, err := rpcParams(mux.MethodPaneWrite, write.rpc.params)
	require.NoError(t, err)
	got, ok := msg.(*schema.WriteRequest)
	require.True(t, ok)
	assert.Equal(t, "p", got.GetId())
	assert.Equal(t, "\x03", got.GetData())
	assert.Equal(t, int32(24), got.GetRows())
	assert.Equal(t, int32(80), got.GetCols())

	watch := cmd.ParseOK[Cmd](t, "rpc", "watch", `{"all":true}`)
	msg, err = rpcParams(mux.MethodWatch, watch.rpc.params)
	require.NoError(t, err)
	assert.True(t, msg.(*schema.WatchRequest).GetAll())

	list := cmd.ParseOK[Cmd](t, "rpc", "pane.list", `{"ok":true}`)
	_, err = rpcParams(mux.MethodPaneList, list.rpc.params)
	require.ErrorIs(t, err, mux.ErrProtocol)

	empty, err := rpcParams(mux.MethodSessionGet, nil)
	require.NoError(t, err)
	assert.Nil(t, empty)

	text, err := payloadJSON(&schema.Envelope{Payload: &schema.Envelope_Ack{Ack: &schema.Ack{Ok: true}}})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, text)
	text, err = payloadJSON(&schema.Envelope{Payload: &schema.Envelope_Session{Session: &schema.SessionInfo{Id: "s"}}})
	require.NoError(t, err)
	assert.Equal(t, `{"id":"s"}`, text)
	text, err = payloadJSON(&schema.Envelope{})
	require.NoError(t, err)
	assert.Equal(t, `{}`, text)
}
