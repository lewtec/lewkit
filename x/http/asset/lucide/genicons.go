//go:build ignore

// genicons writes name.go and icons.gen.go from the Lucide commit in modot.lock.json.
package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"go/format"
	"go/token"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lewtec/lewkit/x/errs"
)

const archiveLimit = 64 << 20

var (
	errModuleRoot = errs.Wrap(errs.ErrMissing, "go.mod")
	errLockJSON   = errs.Wrap(errs.ErrInvalid, "modot.lock.json")
	errLucidePin  = errs.Wrap(errs.ErrMissing, "github:lucide-icons/lucide pin")
	errArchive    = errs.Wrap(errs.ErrUnavailable, "lucide archive")
	errIcons      = errs.Wrap(errs.ErrMissing, "lucide icons")
	errIconName   = errs.Wrap(errs.ErrDuplicate, "lucide icon name")
	errIconFile   = errs.Wrap(errs.ErrInvalid, "lucide icon file")
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "genicons: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	digest, err := lucideDigest(filepath.Join(root, "modot.lock.json"))
	if err != nil {
		return err
	}
	icons, err := fetch(digest)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "x/http/asset/lucide")
	names, err := format.Source(nameSource(icons))
	if err != nil {
		return err
	}
	bodies, err := format.Source(bodySource(icons))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "name.go"), names, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "icons.gen.go"), bodies, 0o644)
}

type icon struct {
	file string
	id   string
	svg  string
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		info, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil && !info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errModuleRoot
		}
		dir = parent
	}
}

func lucideDigest(lockPath string) (string, error) {
	text, err := os.ReadFile(lockPath)
	if err != nil {
		return "", err
	}
	var lock struct {
		Dependencies []struct {
			Kind          string `json:"kind"`
			Ref           string `json:"ref"`
			CurrentValue  string `json:"currentValue"`
			CurrentDigest string `json:"currentDigest"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(text, &lock); err != nil {
		return "", fmt.Errorf("%w: %w", errLockJSON, err)
	}
	for _, dep := range lock.Dependencies {
		if dep.Kind != "source" || dep.Ref != "github:lucide-icons/lucide" {
			continue
		}
		if dep.CurrentValue == "" || !hexDigest(dep.CurrentDigest) {
			return "", errLucidePin
		}
		return dep.CurrentDigest, nil
	}
	return "", errLucidePin
}

func hexDigest(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

func fetch(digest string) ([]icon, error) {
	url := "https://codeload.github.com/lucide-icons/lucide/tar.gz/" + digest
	client := &http.Client{Timeout: 2 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errArchive, err)
	}
	req.Header.Set("User-Agent", "lewkit-genicons")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errArchive, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", errArchive, resp.Status)
	}
	return readIcons(resp.Body)
}

func readIcons(r io.Reader) ([]icon, error) {
	limited := &io.LimitedReader{R: r, N: archiveLimit + 1}
	gz, err := gzip.NewReader(limited)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errArchive, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	seenFile := map[string]struct{}{}
	seenID := map[string]string{}
	var icons []icon
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if limited.N <= 0 {
				return nil, fmt.Errorf("%w: archive exceeds %d bytes", errArchive, archiveLimit)
			}
			return nil, fmt.Errorf("%w: %w", errArchive, err)
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		name := path.Clean(hdr.Name)
		if path.Base(path.Dir(name)) != "icons" || path.Ext(name) != ".svg" {
			continue
		}
		file := strings.TrimSuffix(path.Base(name), ".svg")
		if file == "" || strings.HasPrefix(file, ".") {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errArchive, err)
		}
		if _, ok := seenFile[file]; ok {
			return nil, fmt.Errorf("%w: %s", errIconName, file)
		}
		id, err := goName(file)
		if err != nil {
			return nil, err
		}
		if prev, ok := seenID[id]; ok {
			return nil, fmt.Errorf("%w: %s and %s", errIconName, prev, file)
		}
		seenFile[file] = struct{}{}
		seenID[id] = file
		icons = append(icons, icon{file: file, id: id, svg: svgText(body)})
	}
	if limited.N <= 0 {
		return nil, fmt.Errorf("%w: archive exceeds %d bytes", errArchive, archiveLimit)
	}
	if len(icons) == 0 {
		return nil, errIcons
	}
	sort.Slice(icons, func(i, j int) bool { return icons[i].file < icons[j].file })
	return icons, nil
}

func svgText(raw []byte) string {
	s := strings.ReplaceAll(string(raw), "\r\n", "\n")
	return strings.TrimRight(s, "\r\n")
}

func goName(file string) (string, error) {
	var b strings.Builder
	b.WriteString("Name")
	for _, part := range strings.Split(file, "-") {
		if part == "" {
			return "", fmt.Errorf("%w: %s", errIconFile, file)
		}
		r, size := utf8.DecodeRuneInString(part)
		if r == utf8.RuneError && size == 1 {
			return "", fmt.Errorf("%w: %s", errIconFile, file)
		}
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			b.WriteString(part[size:])
			continue
		}
		b.WriteString(part)
	}
	id := b.String()
	if !token.IsIdentifier(id) || token.IsKeyword(id) {
		return "", fmt.Errorf("%w: %s", errIconFile, file)
	}
	return id, nil
}

func nameSource(icons []icon) []byte {
	var b strings.Builder
	b.WriteString(`// Code generated from the lucide modot input icons/*.svg. DO NOT EDIT.

package lucide

// Name is one icon in this set. The text is the Lucide file name.
//
//lewkit:enum
type Name string

const (
`)
	for _, item := range icons {
		fmt.Fprintf(&b, "\t%s Name = %s\n", item.id, strconv.Quote(item.file))
	}
	b.WriteString(")\n")
	return []byte(b.String())
}

func bodySource(icons []icon) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, `// Code generated from the lucide modot input icons/*.svg. DO NOT EDIT.

package lucide

const iconCount = %d

var iconBody = map[Name]string{
`, len(icons))
	for _, item := range icons {
		fmt.Fprintf(&b, "\t%s: %s,\n", item.id, strconv.Quote(item.svg))
	}
	b.WriteString("}\n")
	return []byte(b.String())
}
