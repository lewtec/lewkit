package prelude

import (
	"io"

	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

var generatedBy = generate.Notice("prelude")

func write(w io.Writer, packageName string, imports []string) error {
	file := golang.NewFile(packageName)
	file.Header = "// " + generatedBy
	for _, path := range imports {
		file.Import(path).Alias("_")
	}
	return file.Make(w)
}
