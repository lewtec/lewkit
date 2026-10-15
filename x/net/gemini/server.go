package gemini

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"time"
)

const maxRequestSize = 1024

// Server answers one Gemini request per connection.
// Handler runs for each request. A nil Handler is [ErrNoHandler].
type Server struct {
	Handler Handler
}

// Listen opens a TLS listener for Gemini.
// An empty addr listens on 127.0.0.1:0. The certificate is ephemeral and self-signed.
func Listen(ctx context.Context, addr string) (net.Listener, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	cert, err := selfSigned()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrServe, err)
	}
	ln, err := tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrServe, err)
	}
	return ln, nil
}

// Serve accepts from ln until ctx is canceled.
func (server *Server) Serve(ctx context.Context, ln net.Listener) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if server == nil || server.Handler == nil {
		return ErrNoHandler
	}
	if ln == nil {
		return ErrNoListener
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = ln.Close()
		case <-done:
		}
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%w: %w", ErrServe, err)
		}
		go server.serve(ctx, conn)
	}
}

func (server *Server) serve(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	}
	line, err := readRequest(conn)
	if err != nil || strings.TrimSpace(line) == "" {
		_ = WriteResponse(conn, Response{Status: CodeBadRequest, Meta: "bad request"})
		return
	}
	target, err := NormalizeURL(line)
	if err != nil {
		_ = WriteResponse(conn, Response{Status: CodeBadRequest, Meta: "bad request"})
		return
	}
	resp, err := server.Handler(ctx, target)
	if err != nil {
		_ = WriteResponse(conn, Response{Status: CodeCGIError, Meta: cleanMeta(err.Error())})
		return
	}
	if resp.Status == 0 {
		resp.Status = CodePermanentFailure
		if resp.Meta == "" {
			resp.Meta = "missing status"
		}
	}
	if Category(resp.Status) == StatusSuccess && len(resp.Body) > maxBodySize {
		resp = Response{Status: CodeTemporaryFailure, Meta: "body too large"}
	}
	_ = WriteResponse(conn, resp)
}

// WriteResponse writes a Gemini status line and, for success, the body.
func WriteResponse(w io.Writer, resp Response) error {
	if _, err := fmt.Fprintf(w, "%02d %s\r\n", resp.Status, cleanMeta(resp.Meta)); err != nil {
		return fmt.Errorf("%w: %w", ErrStatusFailed, err)
	}
	if Category(resp.Status) != StatusSuccess || len(resp.Body) == 0 {
		return nil
	}
	if _, err := w.Write(resp.Body); err != nil {
		return fmt.Errorf("%w: %w", ErrStatusFailed, err)
	}
	return nil
}

func readRequest(r io.Reader) (string, error) {
	buf := make([]byte, 0, 64)
	var one [1]byte
	for {
		if len(buf) >= maxRequestSize {
			return "", ErrBadHeader
		}
		n, err := r.Read(one[:])
		if n == 1 {
			buf = append(buf, one[0])
			if one[0] == '\n' {
				return strings.TrimRight(string(buf), "\r\n"), nil
			}
		}
		if err != nil {
			return "", err
		}
	}
}

func cleanMeta(meta string) string {
	meta = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(meta, "\r", " "), "\n", " "))
	if len(meta) > 1024 {
		meta = meta[:1024]
	}
	return meta
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
