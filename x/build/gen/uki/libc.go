package uki

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
)

// Alpine 3.24 userspace for a dynamic guest. Versions are whatever that
// release currently publishes. musl is the libc. gcompat supplies the
// glibc names a cgo-free Go binary records (libc.so.6, libpthread.so.0,
// ld-linux). libucontext and musl-obstack are gcompat's dependencies.
var alpineLibOrder = []string{
	"musl",
	"gcompat",
	"musl-obstack",
	"libucontext",
}

var (
	libcMu    sync.Mutex
	libcCache = map[string]map[string][]byte{}
)

// fetchAlpineLibs returns guest paths, without a leading slash, of the
// Alpine 3.24 libc packages. A symlink is stored as a copy of its target
// so the initramfs does not need symlink support.
func fetchAlpineLibs(ctx context.Context, goarch string) (map[string][]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	libcMu.Lock()
	defer libcMu.Unlock()
	if files, ok := libcCache[goarch]; ok {
		return files, nil
	}
	if _, err := AlpineArch(goarch); err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	links := map[string]string{}
	for _, pkg := range alpineLibOrder {
		raw, err := readAlpinePkg(ctx, goarch, pkg)
		if err != nil {
			return nil, err
		}
		if err := mergeApk(raw, files, links); err != nil {
			return nil, err
		}
	}
	for name, target := range links {
		body, ok := files[linkTarget(name, target)]
		if !ok {
			continue
		}
		files[name] = body
	}
	// gcompat has no libdl.so.2. The other glibc sonames in that package
	// are libgcompat.so.0, and musl exports dlopen.
	if _, ok := files["lib/libdl.so.2"]; !ok {
		if body, ok := files["lib/libgcompat.so.0"]; ok {
			files["lib/libdl.so.2"] = body
		}
	}
	libcCache[goarch] = files
	return files, nil
}

func readAlpinePkg(ctx context.Context, goarch, name string) ([]byte, error) {
	dst, err := fetchAlpineApk(ctx, goarch, name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(dst)
}

func mergeApk(raw []byte, files map[string][]byte, links map[string]string) error {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := cleanArchivePath(hdr.Name)
		if name == "." || strings.HasPrefix(path.Base(name), ".") {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			body, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			files[name] = body
		case tar.TypeSymlink:
			links[name] = hdr.Linkname
		}
	}
}

func linkTarget(name, target string) string {
	if strings.HasPrefix(target, "/") {
		return cleanArchivePath(target)
	}
	return path.Clean(path.Join(path.Dir(name), target))
}

func cleanArchivePath(name string) string {
	name = strings.TrimPrefix(name, "./")
	name = path.Clean(name)
	return strings.TrimPrefix(name, "/")
}

func sysrootHits(files map[string][]byte, name string) []libHit {
	if strings.HasPrefix(name, "/") {
		guest := cleanArchivePath(name)
		body, ok := files[guest]
		if !ok || !isELF(body) {
			return nil
		}
		return []libHit{{guest: guest, data: body}}
	}
	var hits []libHit
	for guest, body := range files {
		if path.Base(guest) != name || !isELF(body) {
			continue
		}
		hits = append(hits, libHit{guest: guest, data: body})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].guest < hits[j].guest })
	return hits
}

// alpineRuntime is what gcompat's ld-linux stub execs. Those paths are
// not DT_NEEDED entries of the stub.
func alpineRuntime(goarch string) []string {
	switch goarch {
	case "amd64":
		return []string{"libgcompat.so.0", "ld-musl-x86_64.so.1"}
	case "arm64":
		return []string{"libgcompat.so.0", "ld-musl-aarch64.so.1"}
	default:
		return nil
	}
}
