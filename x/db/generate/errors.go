package generate

import "errors"

var (
	errDirRequired     = errors.New("directory required")
	errNoModule        = errors.New("go.mod: no module line")
	errNoGoMod         = errors.New("no go.mod")
	errNoEngines       = errors.New("no sqlite/ or postgres")
	errNoMigrationsDir = errors.New("missing migrations")
	errNoNamedQueries  = errors.New("no named queries")
	errQueryMismatch   = errors.New("query mismatch")
	errQueryRedeclared = errors.New("query redeclared")
	errMethodMismatch  = errors.New("method mismatch")
	errTypeMismatch    = errors.New("type mismatch")
	errSQLC            = errors.New("sqlc generate")
)
