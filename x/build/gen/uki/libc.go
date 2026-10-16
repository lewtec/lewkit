package uki

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Alpine 3.24 userspace for a dynamic guest. The kernel package is the
// same release. musl is the libc. gcompat is the glibc names a cgo-free
// Go binary records (libc.so.6, libpthread.so.0, ld-linux). libucontext
// and musl-obstack are gcompat's own dependencies.
var alpineLibSHA = map[string]map[string]string{
	"amd64": {
		"musl-1.2.6-r2":         "573712e2f49c15bfc20a2699f204acdfc74c772722b15e7353d768057fae0e71",
		"gcompat-1.1.0-r4":      "6f4f58bc5df7c9a6460710e30fd810eabc309cd6b7f209f491a9dfaf44859dbc",
		"musl-obstack-1.2.3-r2": "0bb29f7975b24f18170bf0ccbbf81e85e5ccf05f913544a6d7378f03da535852",
		"libucontext-1.5.1-r0":  "fda972c0297b95efa2801de28d0a2995457cab85401c6d885a9c66641d79500c",
	},
	"arm64": {
		"musl-1.2.6-r2":         "5e9674b7f41152fe2119093b5cb4c13eaaadb19c2d5422b2d7267913e663ee6e",
		"gcompat-1.1.0-r4":      "8e6b58c16e3cceaf714d71d3906f2b6b42fa299027519c5f2ec46960ff2776e2",
		"musl-obstack-1.2.3-r2": "08ac3018b48849dce6737f2eca8d925fb82c6c7cd416e70f8f5975ac97e725cc",
		"libucontext-1.5.1-r0":  "bb8ff2783ebd1488baebd403b048ea62b8d64f166ea402584f01f1b1a6c89d4f",
	},
}

var alpineLibOrder = []string{
	"musl-1.2.6-r2",
	"gcompat-1.1.0-r4",
	"musl-obstack-1.2.3-r2",
	"libucontext-1.5.1-r0",
}

var (
	libcMu    sync.Mutex
	libcCache = map[string]map[string][]byte{}
)

func alpineLibURL(goarch, pkg string) (string, error) {
	arch, err := AlpineArch(goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/main/%s/%s.apk", alpineRelease, arch, pkg), nil
}

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
	shas, ok := alpineLibSHA[goarch]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrArch, goarch)
	}
	files := map[string][]byte{}
	links := map[string]string{}
	for _, pkg := range alpineLibOrder {
		raw, err := readAlpinePkg(ctx, goarch, pkg, shas[pkg])
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
	// gcompat 1.1.0-r4 has no libdl.so.2. The other glibc sonames in that
	// package are libgcompat.so.0, and musl exports dlopen.
	if _, ok := files["lib/libdl.so.2"]; !ok {
		if body, ok := files["lib/libgcompat.so.0"]; ok {
			files["lib/libdl.so.2"] = body
		}
	}
	libcCache[goarch] = files
	return files, nil
}

func readAlpinePkg(ctx context.Context, goarch, pkg, want string) ([]byte, error) {
	arch, err := AlpineArch(goarch)
	if err != nil {
		return nil, err
	}
	dst := filepath.Join(os.TempDir(), "lewkit-alpine", "v"+alpineRelease, arch, pkg+".apk")
	raw, err := os.ReadFile(dst)
	if err != nil || sha256Hex(raw) != want {
		url, err := alpineLibURL(goarch, pkg)
		if err != nil {
			return nil, err
		}
		raw, err = httpGet(ctx, url)
		if err != nil {
			return nil, err
		}
		if got := sha256Hex(raw); got != want {
			return nil, fmt.Errorf("%w: %s %s sha256 %s", errUKI, pkg, goarch, got)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			return nil, err
		}
	}
	return raw, nil
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
