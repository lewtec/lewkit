package applications

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("uv", "astral-sh/uv", tool.Binary("uv"))
	registry.RegisterGitHub("fzf", "junegunn/fzf", tool.Binary("fzf"))
	registry.RegisterGitHub("fd", "sharkdp/fd", tool.Binary("fd"))
	registry.RegisterGitHub("sops", "getsops/sops", tool.Binary("sops"))
	registry.RegisterGitHub("tflint", "terraform-linters/tflint", tool.Binary("tflint"))
	registry.RegisterGitHub("opencode", "anomalyco/opencode", tool.Binary("opencode"))
	registry.RegisterGitHub("rclone", "rclone/rclone", tool.Binary("rclone"))
	registry.RegisterGitHub("rtk", "rtk-ai/rtk", tool.Binary("rtk"))
	registry.RegisterGitHub("resvg", "linebender/resvg", tool.Binary("resvg"))
	registry.RegisterGitHub("codex", "openai/codex", tool.Binary("codex"))
	registry.RegisterGitHub("contapila", "lucasew/contapila", tool.Binary("contapila"))
	registry.RegisterGitHub("docker-compose", "docker/compose", tool.Binary("docker-compose"))
	registry.RegisterGitHub("refactree", "lucasew/refactree", tool.Binary("rft"))
	registry.RegisterGitHub("ripgrep", "burntsushi/ripgrep", tool.Binary("rg"))
	registry.RegisterGitHub("mise", "jdx/mise", tool.Binary("mise"))
	registry.RegisterGitHub("bun", "oven-sh/bun", tool.Binary("bun"))
}
