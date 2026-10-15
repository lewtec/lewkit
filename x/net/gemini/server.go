package gemini

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const maxRequestSize = 1024

// Server answers one Gemini request per connection.
// Addr is the [Server.ListenAndServeTLS] address. Empty means ":1965".
// TLSConfig supplies the certificate for [Server.ServeTLS] and [Server.ListenAndServeTLS].
// Those methods load certFile and keyFile when the config has no certificate.
type Server struct {
	Addr      string
	Handler   Handler
	TLSConfig *tls.Config
}

// ListenAndServeTLS listens on addr and serves Gemini until ctx is canceled.
// certFile and keyFile are passed the way [http.ListenAndServeTLS] passes them.
func ListenAndServeTLS(ctx context.Context, addr, certFile, keyFile string, handler Handler) error {
	return (&Server{Addr: addr, Handler: handler}).ListenAndServeTLS(ctx, certFile, keyFile)
}

// ListenAndServeTLS listens on [Server.Addr] and then calls [Server.ServeTLS].
// An empty Addr uses ":1965".
// certFile and keyFile are required when TLSConfig has no certificate,
// matching [http.Server.ListenAndServeTLS].
func (server *Server) ListenAndServeTLS(ctx context.Context, certFile, keyFile string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if server == nil || server.Handler == nil {
		return ErrNoHandler
	}
	addr := strings.TrimSpace(server.Addr)
	if addr == "" {
		addr = ":1965"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrServe, err)
	}
	defer ln.Close()
	return server.ServeTLS(ctx, ln, certFile, keyFile)
}

// ServeTLS accepts on ln, wraps each connection in TLS, and calls [Server.Serve].
// certFile and keyFile are required when TLSConfig has no certificate,
// matching [http.Server.ServeTLS].
func (server *Server) ServeTLS(ctx context.Context, ln net.Listener, certFile, keyFile string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if server == nil || server.Handler == nil {
		return ErrNoHandler
	}
	if ln == nil {
		return ErrNoListener
	}
	config, err := server.tlsConfig(certFile, keyFile)
	if err != nil {
		return err
	}
	return server.Serve(ctx, tls.NewListener(ln, config))
}

// Serve accepts from ln until ctx is canceled.
// ln already speaks TLS, as with [http.Server.Serve].
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

// tlsConfig clones TLSConfig and installs certFile and keyFile when the
// config has no certificate, or when a file path is set.
func (server *Server) tlsConfig(certFile, keyFile string) (*tls.Config, error) {
	config := &tls.Config{}
	if server != nil && server.TLSConfig != nil {
		config = server.TLSConfig.Clone()
	}
	hasCert := len(config.Certificates) > 0 || config.GetCertificate != nil || config.GetConfigForClient != nil
	if !hasCert || certFile != "" || keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoCertificate, err)
		}
		config.Certificates = []tls.Certificate{cert}
	}
	if config.MinVersion == 0 {
		config.MinVersion = tls.VersionTLS12
	}
	return config, nil
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
