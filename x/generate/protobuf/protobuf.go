// Package protobuf writes Go from a .proto file.
package protobuf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const protobufLazy = "protobuf"

// Run runs protoc --go_out on protoFile. goPackage is the M mapping when
// the proto has no go_package option. protoc comes from workspaced
// lazy_tools.protobuf; the lockfile owns the version.
func Run(ctx context.Context, protoFile, goPackage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if protoFile == "" {
		return errFileRequired
	}
	protoFile, err := filepath.Abs(protoFile)
	if err != nil {
		return err
	}
	if _, err := os.Stat(protoFile); err != nil {
		return err
	}
	plugin, err := exec.CommandContext(ctx, "go", "tool", "-n", "protoc-gen-go").Output()
	if err != nil {
		return fmt.Errorf("%w: %w", errPlugin, err)
	}
	dir := filepath.Dir(protoFile)
	base := filepath.Base(protoFile)
	args := []string{
		"open", "lazy", "--bin", "protoc", protobufLazy, "--",
		"--plugin=protoc-gen-go=" + strings.TrimSpace(string(plugin)),
		"--proto_path=" + dir,
		"--go_out=" + dir,
		"--go_opt=paths=source_relative",
	}
	if goPackage != "" {
		args = append(args, "--go_opt=M"+base+"="+goPackage)
	}
	args = append(args, protoFile)
	cmd := exec.CommandContext(ctx, "go", append([]string{"tool", "workspaced"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %w", errProtoc, err)
	}
	return nil
}
