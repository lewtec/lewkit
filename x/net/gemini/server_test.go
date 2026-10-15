package gemini

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeEnum(t *testing.T) {
	parsed, err := ParseCode("success")
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, parsed)
	assert.Equal(t, Code(20), CodeSuccess)
	assert.Equal(t, "success", CodeSuccess.String())
	assert.Equal(t, "unknown", Code(99).String())
	assert.False(t, Code(99).IsValid())
	assert.Equal(t, StatusSuccess, Category(CodeSuccess))
	assert.Equal(t, StatusRedirect, Category(CodeRedirectPermanent))

	_, err = ParseCode("nope")
	require.ErrorIs(t, err, ErrCode)

	category, err := ParseStatusCategory("input")
	require.NoError(t, err)
	assert.Equal(t, StatusInput, category)
	assert.Equal(t, "input", StatusInput.String())
}

func TestWriteAndReadResponse(t *testing.T) {
	var buf bytes.Buffer
	err := WriteResponse(&buf, Response{
		Status: CodeSuccess,
		Meta:   "text/gemini",
		Body:   []byte("# Hi\n"),
	})
	require.NoError(t, err)
	got, err := ReadResponse(&buf)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, got.Status)
	assert.Equal(t, "text/gemini", got.Meta)
	assert.Equal(t, "# Hi\n", string(got.Body))
}

func TestListenAndServeGuards(t *testing.T) {
	require.Panics(t, func() { _, _ = Listen(nil, "127.0.0.1:0") })
	require.Panics(t, func() { _ = (&Server{}).Serve(nil, nil) })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Listen(ctx, "127.0.0.1:0")
	require.ErrorIs(t, err, context.Canceled)
	err = (&Server{}).Serve(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)

	err = (&Server{}).Serve(t.Context(), nil)
	require.ErrorIs(t, err, ErrNoHandler)

	err = (&Server{Handler: func(context.Context, *url.URL) (Response, error) {
		return Response{}, nil
	}}).Serve(t.Context(), nil)
	require.ErrorIs(t, err, ErrNoListener)
}

func TestClientFetchesServer(t *testing.T) {
	ln, err := Listen(t.Context(), "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	errc := make(chan error, 1)
	go func() {
		errc <- (&Server{Handler: serveCapsule}).Serve(ctx, ln)
	}()

	client := &Client{}
	base := "gemini://" + ln.Addr().String()

	got, err := client.Fetch(t.Context(), base+"/old")
	require.NoError(t, err)
	assert.Equal(t, base+"/new", got.URL)
	assert.Equal(t, CodeSuccess, got.Status)
	assert.Equal(t, "text/gemini", got.Meta)
	assert.Equal(t, "# Arrived\n", string(got.Body))

	missing, err := client.Fetch(t.Context(), base+"/missing")
	require.NoError(t, err)
	assert.Equal(t, CodeNotFound, missing.Status)

	failed, err := client.Fetch(t.Context(), base+"/boom")
	require.NoError(t, err)
	assert.Equal(t, CodeCGIError, failed.Status)

	conn, err := tls.Dial("tcp", ln.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	})
	require.NoError(t, err)
	_, err = io.WriteString(conn, "\r\n")
	require.NoError(t, err)
	bad, err := ReadResponse(conn)
	require.NoError(t, err)
	assert.Equal(t, CodeBadRequest, bad.Status)
	require.NoError(t, conn.Close())

	cancel()
	require.ErrorIs(t, <-errc, context.Canceled)
}

func serveCapsule(_ context.Context, target *url.URL) (Response, error) {
	switch target.Path {
	case "/old":
		return Response{Status: CodeRedirectPermanent, Meta: "/new"}, nil
	case "/new":
		return Response{Status: CodeSuccess, Meta: "text/gemini", Body: []byte("# Arrived\n")}, nil
	case "/boom":
		return Response{}, errors.New("capsule failed")
	default:
		return Response{Status: CodeNotFound, Meta: "not found"}, nil
	}
}
