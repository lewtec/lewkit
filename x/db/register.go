package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"sync"
)

// Connector opens one URL scheme and runs its migrations.
type Connector struct {
	Driver string
	DSN    func(raw string) string
	Up     func(*sql.DB, fs.FS) error
}

var engines sync.Map // string → Connector

// Register installs a connector for a URL scheme (sqlite, postgres, …).
func Register(scheme string, c Connector) {
	engines.Store(strings.ToLower(scheme), c)
}

// Scheme is the engine name for a database URL.
func Scheme(raw string) string {
	scheme, _ := splitURL(raw)
	return scheme
}

func lookup(raw string) (Connector, string, error) {
	scheme, dsn := splitURL(raw)
	if scheme == "" {
		return Connector{}, "", fmt.Errorf("%w: empty url", ErrUnknownScheme)
	}
	v, ok := engines.Load(scheme)
	if !ok {
		return Connector{}, "", fmt.Errorf("%w: %s", ErrUnknownScheme, scheme)
	}
	c := v.(Connector)
	if c.DSN != nil {
		dsn = c.DSN(raw)
	}
	return c, dsn, nil
}

func splitURL(raw string) (scheme, dsn string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if raw == ":memory:" || strings.HasPrefix(raw, "file::memory:") {
		return "sqlite", raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return "sqlite", raw
	}
	scheme = strings.ToLower(u.Scheme)
	switch scheme {
	case "sqlite", "sqlite3", "file":
		return "sqlite", sqliteDSN(u, raw)
	case "postgres", "postgresql":
		return "postgres", raw
	default:
		return scheme, raw
	}
}

func sqliteDSN(u *url.URL, raw string) string {
	if u.Scheme == "file" {
		if u.Opaque != "" {
			return u.Opaque + q(u)
		}
		if u.Path != "" {
			return u.Path + q(u)
		}
		return raw
	}
	if u.Opaque != "" {
		return u.Opaque + q(u)
	}
	if u.Host != "" && u.Path != "" {
		return u.Host + u.Path + q(u)
	}
	if u.Host != "" {
		return u.Host + q(u)
	}
	if u.Path != "" {
		return u.Path + q(u)
	}
	return raw
}

func q(u *url.URL) string {
	if u.RawQuery == "" {
		return ""
	}
	return "?" + u.RawQuery
}
