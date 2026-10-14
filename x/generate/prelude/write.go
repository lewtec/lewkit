package prelude

import (
	"github.com/dave/jennifer/jen"
	"github.com/lewtec/lewkit/x/generate"
)

var generatedBy = generate.Notice("prelude")

func jenFile(packageName string, imports []string) *jen.File {
	file := jen.NewFile(packageName)
	file.HeaderComment(generatedBy)
	for _, imp := range imports {
		file.Anon(imp)
	}
	return file
}
