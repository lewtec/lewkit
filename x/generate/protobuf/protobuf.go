// Package protobuf writes Go from a .proto file.
package protobuf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/tool"

	_ "github.com/lewtec/lewkit/x/tool/github"
)

const protocRef = "github:protocolbuffers/protobuf"

// Run runs protoc --go_out on protoFile. goPackage is the M mapping when
// the proto has no go_package option. protoc is installed with x/tool.
// A workspaced.lock.json above protoFile pins the version; otherwise the
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
	plugin, err := exec.CommandContext(ctx, "go", "tool", "-n", "protoc-gen-go").Output()
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
	command := exec.CommandContext(ctx, protoc, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
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
	store, err := tool.Open(filepath.Join(cache, "lewkit", "tools"))
	if err != nil {
		return "", err
	}
	return store.Ensure(ctx, spec, "protoc")
}

func protocSpec(directory string) (string, error) {
	for {
		spec, found, err := protocSpecIn(directory)
		if err != nil {
			return "", err
		}
		if found {
			return spec, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return protocRef + "@latest", nil
		}
		directory = parent
	}
}

func protocSpecIn(directory string) (string, bool, error) {
	root, err := lewpath.Open(directory)
	if err != nil {
		return "", false, nil
	}
	defer root.Close()
	name := lewpath.New("workspaced.lock.json")
	isFile, err := name.IsFile(root)
	if err != nil || !isFile {
		return "", false, err
	}
	body, err := name.ReadFile(root)
	if err != nil {
		return "", false, err
	}
	var lock struct {
		Dependencies []struct {
			Kind         string `json:"kind"`
			Ref          string `json:"ref"`
			CurrentValue string `json:"currentValue"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		return "", false, fmt.Errorf("workspaced.lock.json: %w", err)
	}
	for _, dependency := range lock.Dependencies {
		if dependency.Kind == "tool" && dependency.Ref == protocRef && dependency.CurrentValue != "" {
			return protocRef + "@" + dependency.CurrentValue, true, nil
		}
	}
	return "", false, nil
}
