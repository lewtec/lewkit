// Package generate is the lewkit generate command.
// cmd/lewkit stores Cmd and does not declare the subcommands.
package generate

// Cmd is lewkit generate.
type Cmd struct {
	db       *dbCmd
	prelude  *preludeCmd
	protobuf *protobufCmd
	shader   *shaderCmd
	header   *headerCmd
	clib     *clibCmd
	enum     *enumCmd
}

// Description is the command summary.
func (Cmd) Description() string {
	return "generate code"
}
