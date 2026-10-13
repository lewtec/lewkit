package gotool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runtimeDeps locates the installed runtime archives. The compiler does
// not build the standard library. compileCfg is the importcfg for package
// main. linkDeps is the packagefile lines for the link.
func runtimeDeps(dir string) (compileCfg, linkDeps string, err error) {
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	if _, err = os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tool\n\ngo 1.27\n"), 0o644)
		if err != nil {
			return "", "", err
		}
	}
	depsPath := filepath.Join(dir, "deps")
	compileCfg = filepath.Join(dir, "importcfg")
	if body, readErr := os.ReadFile(depsPath); readErr == nil {
		return compileCfg, string(body), nil
	}
	cmd := exec.Command("go", "list", "-deps", "-export", "-f", "{{if .Export}}{{.ImportPath}} {{.Export}}{{end}}", "runtime")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("go list runtime: %w\n%s", err, stderr.String())
	}
	var runtimeArchive string
	var b strings.Builder
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path, file, ok := strings.Cut(line, " ")
		if !ok {
			return "", "", fmt.Errorf("go list line %q", line)
		}
		fmt.Fprintf(&b, "packagefile %s=%s\n", path, file)
		if path == "runtime" {
			runtimeArchive = file
		}
	}
	if runtimeArchive == "" {
		return "", "", fmt.Errorf("runtime has no export archive")
	}
	if err = os.WriteFile(compileCfg, []byte("# import config\npackagefile runtime="+runtimeArchive+"\n"), 0o644); err != nil {
		return "", "", err
	}
	if err = os.WriteFile(depsPath, []byte(b.String()), 0o644); err != nil {
		return "", "", err
	}
	return compileCfg, b.String(), nil
}
