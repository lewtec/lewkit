// Package protobuf writes Go from a .proto file.
package protobuf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/tool"
	_ "github.com/lewtec/lewkit/x/tool/prelude"
)

const protocRef = "github:protocolbuffers/protobuf"

// Run runs protoc --go_out on protoFile. goPackage is the M mapping when
// the proto has no go_package option. protoc is installed with x/tool.
// A modot.lock.json above protoFile pins the version; otherwise the
// spec is latest.
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
	plugin, err := execdriver.Output(ctx, execdriver.MustCommand(ctx, "go", "tool", "-n", "protoc-gen-go"))
	if err != nil {
		return fmt.Errorf("%w: %w", errPlugin, err)
	}
	protoc, err := ensureProtoc(ctx, filepath.Dir(protoFile))
	if err != nil {
		return fmt.Errorf("%w: %w", errProtoc, err)
	}
	dir := filepath.Dir(protoFile)
	base := filepath.Base(protoFile)
	args := []string{
		"--plugin=protoc-gen-go=" + strings.TrimSpace(string(plugin)),
		"--proto_path=" + dir,
		"--go_out=" + dir,
		"--go_opt=paths=source_relative",
	}
	if goPackage != "" {
		args = append(args, "--go_opt=M"+base+"="+goPackage)
	}
	args = append(args, protoFile)
	command := execdriver.MustCommand(ctx, protoc, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := execdriver.Run(ctx, command); err != nil {
		return fmt.Errorf("%w: %w", errProtoc, err)
	}
	return nil
}

func ensureProtoc(ctx context.Context, directory string) (string, error) {
	spec, err := protocSpec(directory)
	if err != nil {
		return "", err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	store, err := tool.Open(filepath.Join(cache, release.Name(), "tools"))
	if err != nil {
		return "", err
	}
	return store.Ensure(ctx, spec, "protoc")
}

func protocSpec(directory string) (string, error) {
	return tool.LockSpec(directory, protocRef)
}
