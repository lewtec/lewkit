package dockercompose

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("docker-compose", "docker/compose", tool.Binary("docker-compose"))
}
