package uki

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const alpineRelease = "3.24"

// alpinePackage is one package in an Alpine APKINDEX.
// checksum is the SHA1 of the control gzip stream (the index C:Q1 field).
// size is the published apk length.
type alpinePackage struct {
	name     string
	version  string
	repo     string
	size     int64
	checksum []byte
}

// Arch is an Alpine Linux apk architecture.
//
//lewkit:enum
type Arch string

const (
	ArchX86_64  Arch = "x86_64"  // amd64
	ArchAArch64 Arch = "aarch64" // arm64
)

// AlpineArch is the apk architecture for a Go GOARCH.
// amd64 is x86_64. arm64 is aarch64.
func AlpineArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return string(ArchX86_64), nil
	case "arm64":
		return string(ArchAArch64), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrArch, goarch)
	}
}

var (
	alpineIndexMu sync.Mutex
	alpineIndexes = map[string]map[string]alpinePackage{}
)

// fetchKernel downloads the linux-virt apk Alpine currently publishes for goarch.
// A file already in the cache is reused when it still matches the index.
func fetchKernel(ctx context.Context, goarch string) (string, error) {
	return fetchAlpineApk(ctx, goarch, "linux-virt")
}

func fetchAlpineApk(ctx context.Context, goarch, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	arch, err := AlpineArch(goarch)
	if err != nil {
		return "", err
	}
	pkg, err := lookupAlpine(ctx, arch, name)
	if err != nil {
		return "", err
	}
	dst := filepath.Join(os.TempDir(), "lewkit-alpine", "v"+alpineRelease, pkg.repo, arch, pkg.name+"-"+pkg.version+".apk")
	if verifyAlpineApk(dst, pkg) == nil {
		return dst, nil
	}
	if err := downloadAlpineApk(ctx, arch, pkg, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func lookupAlpine(ctx context.Context, arch, name string) (alpinePackage, error) {
	for _, repo := range []string{"main", "community"} {
		index, err := alpineIndex(ctx, arch, repo)
		if err != nil {
			return alpinePackage{}, err
		}
		pkg, ok := index[name]
		if !ok {
			continue
		}
		if pkg.version == "" || pkg.size <= 0 || len(pkg.checksum) != sha1.Size {
			return alpinePackage{}, fmt.Errorf("%w: %s", errUKI, name)
		}
		pkg.repo = repo
		return pkg, nil
	}
	return alpinePackage{}, fmt.Errorf("%w: %s for %s", errAlpinePackage, name, arch)
}

func alpineIndex(ctx context.Context, arch, repo string) (map[string]alpinePackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := arch + "/" + repo
	alpineIndexMu.Lock()
	defer alpineIndexMu.Unlock()
	if index, ok := alpineIndexes[key]; ok {
		return index, nil
	}
	raw, err := httpGet(ctx, alpineIndexURL(repo, arch))
	if err != nil {
		return nil, err
	}
	index, err := readAPKINDEX(raw)
	if err != nil {
		return nil, err
	}
	alpineIndexes[key] = index
	return index, nil
}

func alpineIndexURL(repo, arch string) string {
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/%s/%s/APKINDEX.tar.gz", alpineRelease, repo, arch)
}

func alpineApkURL(repo, arch string, pkg alpinePackage) string {
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/%s/%s/%s-%s.apk", alpineRelease, repo, arch, pkg.name, pkg.version)
}

func downloadAlpineApk(ctx context.Context, arch string, pkg alpinePackage, dst string) error {
	url := alpineApkURL(pkg.repo, arch, pkg)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s: %s", errUKI, url, resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	if err := verifyAlpineApk(tmp, pkg); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// verifyAlpineApk checks an apk v2 file against the index.
// The file is three gzip streams: signature, control, data.
// C:Q1 is the SHA1 of the control stream, not of the whole file.
// PKGINFO datahash is the SHA256 of the data stream.
func verifyAlpineApk(path string, pkg alpinePackage) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() != pkg.size {
		return fmt.Errorf("%w: %s size %d want %d", errUKI, pkg.name, info.Size(), pkg.size)
	}
	cr := &gzipHashReader{r: bufio.NewReader(f)}
	if err := discardGzip(cr); err != nil {
		return err
	}
	cr.sum = sha1.New()
	control, err := readGzipMember(cr)
	if err != nil {
		return err
	}
	if !bytes.Equal(cr.sum.Sum(nil), pkg.checksum) {
		return fmt.Errorf("%w: %s control checksum", errUKI, pkg.name)
	}
	wantData, err := pkginfoDataHash(control)
	if err != nil {
		return err
	}
	cr.sum = sha256.New()
	if err := discardGzip(cr); err != nil {
		return err
	}
	got := hex.EncodeToString(cr.sum.Sum(nil))
	if got != wantData {
		return fmt.Errorf("%w: %s data checksum", errUKI, pkg.name)
	}
	if _, err := cr.r.ReadByte(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %s trailing bytes", errUKI, pkg.name)
	}
	return nil
}

type gzipHashReader struct {
	r   *bufio.Reader
	sum hash.Hash
}

func (g *gzipHashReader) Read(p []byte) (int, error) {
	n, err := g.r.Read(p)
	if n > 0 && g.sum != nil {
		g.sum.Write(p[:n])
	}
	return n, err
}

func (g *gzipHashReader) ReadByte() (byte, error) {
	b, err := g.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if g.sum != nil {
		g.sum.Write([]byte{b})
	}
	return b, nil
}

func discardGzip(r io.Reader) error {
	n, err := copyGzipMember(r, io.Discard)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: empty apk gzip", errUKI)
	}
	return nil
}

func readGzipMember(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := copyGzipMember(r, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func copyGzipMember(r io.Reader, dst io.Writer) (int64, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return 0, fmt.Errorf("%w: apk gzip: %w", errUKI, err)
	}
	zr.Multistream(false)
	n, err := io.Copy(dst, zr)
	if err != nil {
		return n, fmt.Errorf("%w: apk gzip: %w", errUKI, err)
	}
	if err := zr.Close(); err != nil {
		return n, fmt.Errorf("%w: apk gzip: %w", errUKI, err)
	}
	return n, nil
}

func pkginfoDataHash(control []byte) (string, error) {
	tr := tar.NewReader(bytes.NewReader(control))
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%w: apk has no datahash", errUKI)
		}
		if err != nil {
			return "", fmt.Errorf("%w: apk control: %w", errUKI, err)
		}
		if path.Base(hdr.Name) != ".PKGINFO" {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			key, val, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(key) != "datahash" {
				continue
			}
			sum := strings.TrimSpace(val)
			if len(sum) != sha256.Size*2 {
				return "", fmt.Errorf("%w: apk datahash", errUKI)
			}
			return sum, nil
		}
	}
}

func readAPKINDEX(raw []byte) (map[string]alpinePackage, error) {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: apk index: %w", errUKI, err)
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w: apk index has no APKINDEX", errUKI)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: apk index: %w", errUKI, err)
		}
		if path.Base(hdr.Name) != "APKINDEX" {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		return parseAPKINDEX(string(body))
	}
}

func parseAPKINDEX(text string) (map[string]alpinePackage, error) {
	out := map[string]alpinePackage{}
	for _, rec := range strings.Split(text, "\n\n") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		var name, version, checksum string
		var size int64
		for _, line := range strings.Split(rec, "\n") {
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			switch key {
			case "P":
				name = val
			case "V":
				version = val
			case "C":
				checksum = val
			case "S":
				n, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("%w: apk index size %q", errUKI, val)
				}
				size = n
			}
		}
		if name == "" || version == "" {
			continue
		}
		if _, ok := out[name]; ok {
			continue
		}
		pkg := alpinePackage{name: name, version: version, size: size}
		if checksum != "" {
			sum, err := decodeAlpineChecksum(checksum)
			if err != nil {
				return nil, err
			}
			pkg.checksum = sum
		}
		out[name] = pkg
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: apk index is empty", errUKI)
	}
	return out, nil
}

func decodeAlpineChecksum(field string) ([]byte, error) {
	if !strings.HasPrefix(field, "Q1") {
		return nil, fmt.Errorf("%w: apk checksum %q", errUKI, field)
	}
	sum, err := base64.StdEncoding.DecodeString(field[2:])
	if err != nil || len(sum) != sha1.Size {
		return nil, fmt.Errorf("%w: apk checksum %q", errUKI, field)
	}
	return sum, nil
}
