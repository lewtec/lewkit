package gemini

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	require.Panics(t, func() { _ = ListenAndServeTLS(nil, "127.0.0.1:0", "", "", serveCapsule) })
	require.Panics(t, func() { _ = (&Server{}).ServeTLS(nil, nil, "", "") })
	require.Panics(t, func() { _ = (&Server{}).Serve(nil, nil) })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := (&Server{Handler: serveCapsule}).ListenAndServeTLS(ctx, "", "")
	require.ErrorIs(t, err, context.Canceled)
	err = (&Server{Handler: serveCapsule}).ServeTLS(ctx, nil, "", "")
	require.ErrorIs(t, err, context.Canceled)
	err = (&Server{}).Serve(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)

	err = (&Server{}).Serve(t.Context(), nil)
	require.ErrorIs(t, err, ErrNoHandler)
	err = (&Server{Handler: serveCapsule}).Serve(t.Context(), nil)
	require.ErrorIs(t, err, ErrNoListener)
	err = (&Server{Handler: serveCapsule}).ServeTLS(t.Context(), nil, "", "")
	require.ErrorIs(t, err, ErrNoListener)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	err = (&Server{Handler: serveCapsule}).ServeTLS(t.Context(), ln, "", "")
	require.ErrorIs(t, err, ErrNoCertificate)
}

func TestClientFetchesServer(t *testing.T) {
	cert, err := selfSigned()
	require.NoError(t, err)
	t.Run("config", func(t *testing.T) {
		fetchCapsule(t, &Server{
			Handler:   serveCapsule,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
		}, "", "")
	})
	t.Run("files", func(t *testing.T) {
		certFile, keyFile := writeKeyPair(t, cert)
		fetchCapsule(t, &Server{Handler: serveCapsule}, certFile, keyFile)
	})
}

func fetchCapsule(t *testing.T, server *Server, certFile, keyFile string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	errc := make(chan error, 1)
	go func() {
		errc <- server.ServeTLS(ctx, ln, certFile, keyFile)
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

func selfSigned() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func writeKeyPair(t *testing.T, cert tls.Certificate) (string, string) {
	t.Helper()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, os.WriteFile(certFile, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0o600))
	return certFile, keyFile
}
