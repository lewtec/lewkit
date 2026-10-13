package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/generate/protobuf"
)

type protobufCmd struct {
	proto     cmd.StringArg `help:".proto file"`
	goPackage cmd.StringArg `long:"package" default:"" help:"Go import path when the proto has no go_package"`
}

func (protobufCmd) Description() string {
	return "generate Go from a .proto file"
}

func (c *protobufCmd) Run(ctx context.Context) error {
	return protobuf.Run(ctx, c.proto.Value(), c.goPackage.Value())
}
