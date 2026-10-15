package gemini

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/errs"
)

const (
	DefaultPort   = "1965"
	maxHeaderSize = 1024 + 8
	maxBodySize   = 4 * 1024 * 1024
	maxRedirects  = 5
)

var (
	ErrInvalidURL    = errs.Wrap(errs.ErrInvalid, "invalid gemini url")
	ErrBadHeader     = errs.Wrap(errs.ErrSyntax, "invalid gemini response header")
	ErrTooManyHops   = errs.Wrap(errs.ErrOutOfRange, "too many gemini redirects")
	ErrStatusFailed  = errs.Wrap(errs.ErrUnavailable, "gemini status error")
	ErrServe         = errs.Wrap(errs.ErrUnavailable, "gemini serve")
	ErrNoHandler     = errs.Wrap(errs.ErrMissing, "gemini handler")
	ErrNoListener    = errs.Wrap(errs.ErrMissing, "gemini listener")
	ErrNoCertificate = errs.Wrap(errs.ErrMissing, "gemini certificate")
)

// StatusCategory classifies a two-digit Gemini status code by its leading digit.
//
//lewkit:enum
type StatusCategory int

const (
	StatusInput       StatusCategory = iota + 1 // input
	StatusSuccess                               // success
	StatusRedirect                              // redirect
	StatusTempFailure                           // temporary
	StatusPermFailure                           // permanent
	StatusClientCert                            // certificate
)

// Response is a parsed Gemini protocol response.
type Response struct {
	URL    string
	Status Code
	Meta   string
	Body   []byte
}

// Handler serves a Gemini request in memory or over a custom transport.
type Handler func(ctx context.Context, target *url.URL) (Response, error)

// Client fetches gemini:// resources over TLS or a local [Handler] table.
type Client struct {
	Timeout  time.Duration
	Hosts    map[string]Handler
	Fallback Handler
}

// NormalizeURL parses and normalizes a Gemini URL string, defaulting the scheme to gemini://.
func NormalizeURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: empty url", ErrInvalidURL)
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "gemini://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "gemini"
	}
	if parsed.Scheme != "gemini" && parsed.Scheme != "about" {
		return nil, fmt.Errorf("%w: scheme %q", ErrInvalidURL, parsed.Scheme)
	}
	if parsed.Scheme == "gemini" && parsed.Host == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	if parsed.Path == "" && parsed.Scheme == "gemini" {
		parsed.Path = "/"
	}
	return parsed, nil
}

// ResolveURL resolves ref relative to baseURL.
func ResolveURL(baseURL, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("%w: empty reference", ErrInvalidURL)
	}
	if strings.Contains(ref, "://") {
		u, err := NormalizeURL(ref)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	}
	base, err := NormalizeURL(baseURL)
	if err != nil {
		return "", err
	}
	rel, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	resolved := base.ResolveReference(rel)
	if resolved.Path == "" && resolved.Scheme == "gemini" {
		resolved.Path = "/"
	}
	return resolved.String(), nil
}

// Fetch retrieves rawURL, following up to 5 redirects.
func (client *Client) Fetch(ctx context.Context, rawURL string) (Response, error) {
	current := rawURL
	for range maxRedirects {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		target, err := NormalizeURL(current)
		if err != nil {
			return Response{}, err
		}
		resp, err := client.fetchOnce(ctx, target)
		if err != nil {
			return Response{}, err
		}
		resp.URL = target.String()
		if Category(resp.Status) == StatusRedirect && strings.TrimSpace(resp.Meta) != "" {
			next, err := ResolveURL(resp.URL, resp.Meta)
			if err != nil {
				return resp, err
			}
			current = next
			continue
		}
		return resp, nil
	}
	return Response{}, fmt.Errorf("%w: %s", ErrTooManyHops, rawURL)
}

func (client *Client) fetchOnce(ctx context.Context, target *url.URL) (Response, error) {
	if client != nil && client.Hosts != nil {
		if handler, ok := client.Hosts[strings.ToLower(target.Hostname())]; ok && handler != nil {
			return handler(ctx, target)
		}
	}
	if client != nil && client.Fallback != nil {
		return client.Fallback(ctx, target)
	}
	return dialGemini(ctx, target, client.timeout())
}

func (client *Client) timeout() time.Duration {
	if client == nil || client.Timeout <= 0 {
		return 10 * time.Second
	}
	return client.Timeout
}

func dialGemini(ctx context.Context, target *url.URL, timeout time.Duration) (Response, error) {
	host := target.Hostname()
	port := target.Port()
	if port == "" {
		port = DefaultPort
	}
	addr := net.JoinHostPort(host, port)
	dialer := &net.Dialer{Timeout: timeout}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true, // Gemini uses TOFU / self-signed certificates in the wild.
		MinVersion:         tls.VersionTLS12,
	})
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrStatusFailed, err)
	}
	defer tlsConn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = tlsConn.SetDeadline(deadline)
	} else {
		_ = tlsConn.SetDeadline(time.Now().Add(timeout))
	}
	if _, err := io.WriteString(tlsConn, target.String()+"\r\n"); err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrStatusFailed, err)
	}
	return ReadResponse(tlsConn)
}

// ReadResponse parses a raw Gemini protocol stream (status header + optional body).
func ReadResponse(r io.Reader) (Response, error) {
	br := bufio.NewReaderSize(r, 16*1024)
	headerLine, err := br.ReadString('\n')
	if err != nil && headerLine == "" {
		return Response{}, fmt.Errorf("%w: %w", ErrBadHeader, err)
	}
	if len(headerLine) > maxHeaderSize {
		return Response{}, fmt.Errorf("%w: header too long", ErrBadHeader)
	}
	headerLine = strings.TrimRight(headerLine, "\r\n")
	if len(headerLine) < 2 {
		return Response{}, fmt.Errorf("%w: %q", ErrBadHeader, headerLine)
	}
	code, err := strconv.Atoi(headerLine[:2])
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrBadHeader, err)
	}
	meta := ""
	if len(headerLine) > 2 {
		if headerLine[2] != ' ' && headerLine[2] != '\t' {
			return Response{}, fmt.Errorf("%w: missing space after status in %q", ErrBadHeader, headerLine)
		}
		meta = strings.TrimSpace(headerLine[3:])
	}
	resp := Response{Status: Code(code), Meta: meta}
	if Category(Code(code)) == StatusSuccess {
		body, err := io.ReadAll(io.LimitReader(br, maxBodySize))
		if err != nil {
			return Response{}, fmt.Errorf("%w: %w", ErrStatusFailed, err)
		}
		resp.Body = body
	}
	return resp, nil
}
