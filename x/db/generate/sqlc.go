package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	sqlc "github.com/sqlc-dev/sqlc/pkg/cli"
	"go.yaml.in/yaml/v3"
)

type sqlcFile struct {
	Version string    `yaml:"version"`
	SQL     []sqlcSQL `yaml:"sql"`
}

type sqlcSQL struct {
	Engine  string   `yaml:"engine"`
	Schema  string   `yaml:"schema"`
	Queries []string `yaml:"queries"`
	Gen     sqlcGen  `yaml:"gen"`
}

type sqlcGen struct {
	Go sqlcGo `yaml:"go"`
}

type sqlcGo struct {
	Package       string `yaml:"package"`
	Out           string `yaml:"out"`
	SQLPackage    string `yaml:"sql_package"`
	EmitInterface bool   `yaml:"emit_interface"`
}

func writeSQLC(root string, engines []engine) error {
	cfg := sqlcFile{Version: "2"}
	for _, e := range engines {
		cfg.SQL = append(cfg.SQL, sqlcSQL{
			Engine:  e.sqlc,
			Schema:  filepath.ToSlash(filepath.Join(e.dir, "migrations")),
			Queries: e.files,
			Gen: sqlcGen{Go: sqlcGo{
				Package:       e.dir,
				Out:           e.dir,
				SQLPackage:    "database/sql",
				EmitInterface: true,
			}},
		})
	}
	b, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "sqlc.yaml"), b, 0o644)
}

func runSQLC(ctx context.Context, root string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg := filepath.Join(root, "sqlc.yaml")
	done := make(chan int, 1)
	go func() {
		done <- sqlc.Run([]string{"generate", "-f", cfg})
	}()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case code := <-done:
		if code != 0 {
			return fmt.Errorf("%w: exit %d", errSQLC, code)
		}
		return nil
	}
}
