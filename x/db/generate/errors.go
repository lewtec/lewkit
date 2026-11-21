package generate

import (
	"errors"

	xgenerate "github.com/lewtec/lewkit/x/generate"
)

var (
	errDirRequired     = xgenerate.ErrDirRequired
	errNoGoMod         = xgenerate.ErrNoGoMod
	errNoEngines       = errors.New("no sqlite/ or postgres")
	errNoMigrationsDir = errors.New("missing migrations")
	errNoNamedQueries  = errors.New("no named queries")
	errQueryMismatch   = errors.New("query mismatch")
	errQueryRedeclared = errors.New("query redeclared")
	errMethodMismatch  = errors.New("method mismatch")
	errTypeMismatch    = errors.New("type mismatch")
	errSQLC            = errors.New("sqlc generate")
)
