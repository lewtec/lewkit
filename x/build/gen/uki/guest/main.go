// Guest init for a uki image. It mounts proc and sys, loads the modules
// listed in load.order, then replaces itself with /app.
package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	_ = os.MkdirAll("/dev", 0o755)
	_ = os.MkdirAll("/proc", 0o755)
	_ = os.MkdirAll("/sys", 0o755)
	_ = os.MkdirAll("/tmp", 0o755)
	if err := unix.Mount("devtmpfs", "/dev", "devtmpfs", 0, ""); err != nil && !errors.Is(err, unix.EBUSY) {
		fmt.Fprintf(os.Stderr, "uki: mount dev: %v\n", err)
	}
	if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Fprintf(os.Stderr, "uki: mount proc: %v\n", err)
	}
	if err := unix.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		fmt.Fprintf(os.Stderr, "uki: mount sys: %v\n", err)
	}
	loadModules("/lib/modules")
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat("/dev/fb0"); err == nil {
			fmt.Fprintf(os.Stderr, "uki: fb0 ready\n")
			break
		}
		if time.Now().After(deadline) {
			entries, _ := os.ReadDir("/dev")
			fmt.Fprintf(os.Stderr, "uki: no /dev/fb0 (%d devices)\n", len(entries))
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	env := append(os.Environ(), "HOME=/tmp", "TERM=linux", "XDG_RUNTIME_DIR=/tmp")
	if st, err := os.Stat("/app"); err != nil {
		fmt.Fprintf(os.Stderr, "uki: app: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "uki: app mode %o size %d\n", st.Mode(), st.Size())
	}
	err := unix.Exec("/app", []string{"/app"}, env)
	fmt.Fprintf(os.Stderr, "uki: exec: %v\n", err)
	for {
		time.Sleep(time.Hour)
	}
}

func loadModules(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "uki: modules: %v\n", err)
		return
	}
	var base string
	for _, entry := range entries {
		if entry.IsDir() {
			base = filepath.Join(root, entry.Name())
			break
		}
	}
	if base == "" {
		fmt.Fprintf(os.Stderr, "uki: modules: empty\n")
		return
	}
	raw, err := os.ReadFile(filepath.Join(base, "load.order"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "uki: load.order: %v\n", err)
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		img, err := inflate(filepath.Join(base, line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "uki: %s: %v\n", line, err)
			continue
		}
		if err := unix.InitModule(img, ""); err != nil && !errors.Is(err, unix.EEXIST) {
			fmt.Fprintf(os.Stderr, "uki: insmod %s: %v\n", line, err)
		}
	}
}

func inflate(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(raw, []byte{0x1f, 0x8b}) {
		return raw, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
