package applications

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("protobuf", "protocolbuffers/protobuf", tool.Binary("protoc"))
}
