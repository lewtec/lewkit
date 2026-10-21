//go:build ignore

// genicons writes name.go and icons.gen.go from the Lucide commit in modot.lock.json.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/tar"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/path/pick"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

const (
	archiveLimit    = 64 << 20
	generatedHeader = "// Code generated from the lucide modot input icons/*.svg. DO NOT EDIT."
	nameComment     = "Name is one icon in this set. The text is the Lucide file name.\n//\n//lewkit:enum"
)

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
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "genicons: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dir, err := moduleRoot()
	if err != nil {
		return err
	}
	root, err := path.Open(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	digest, err := lucideDigest(root)
	if err != nil {
		return err
	}
	icons, err := fetch(ctx, digest)
	if err != nil {
		return err
	}
	if err := writeGo(root, "x/http/asset/lucide/name.go", nameFile(icons)); err != nil {
		return err
	}
	return writeGo(root, "x/http/asset/lucide/icons.gen.go", bodyFile(icons))
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

func lucideDigest(root *path.Root) (string, error) {
	text, err := path.New("modot.lock.json").ReadFile(root)
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

func fetch(ctx context.Context, digest string) ([]icon, error) {
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
	return readIcons(ctx, resp.Body)
}

func readIcons(ctx context.Context, r io.Reader) ([]icon, error) {
	limited := &io.LimitedReader{R: r, N: archiveLimit + 1}
	archive, err := tar.Open(ctx, limited)
	if err != nil {
		if limited.N <= 0 {
			return nil, fmt.Errorf("%w: archive exceeds %d bytes", errArchive, archiveLimit)
		}
		return nil, fmt.Errorf("%w: %w", errArchive, err)
	}
	if limited.N <= 0 {
		return nil, fmt.Errorf("%w: archive exceeds %d bytes", errArchive, archiveLimit)
	}
	stripped, err := fs.StripTopDirectory(archive)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errArchive, err)
	}
	seenFile := map[string]struct{}{}
	seenID := map[string]string{}
	var icons []icon
	for member, err := range fs.Walk(ctx, stripped, pick.Glob("icons/*.svg")) {
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errArchive, err)
		}
		if !member.Mode.IsRegular() {
			continue
		}
		file := member.Name.Stem()
		if file == "" || strings.HasPrefix(file, ".") {
			continue
		}
		body, err := io.ReadAll(member.Reader)
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

func nameFile(icons []icon) *golang.File {
	file := golang.NewFile("lucide")
	file.Header = generatedHeader
	file.Type().Id("Name").Of(golang.Named("string")).Comment(nameComment)
	group := file.Const()
	for _, item := range icons {
		group.Id(item.id).Type(golang.Named("Name")).Values(golang.Lit(item.file))
	}
	return file
}

func bodyFile(icons []icon) *golang.File {
	file := golang.NewFile("lucide")
	file.Header = generatedHeader
	file.Const().Id("iconCount").Values(golang.Lit(len(icons)))
	elts := make([]*golang.Element, 0, len(icons))
	for _, item := range icons {
		elts = append(elts, golang.KeyValue(golang.Id(item.id), golang.Lit(item.svg)))
	}
	file.Var().Id("iconBody").Values(golang.Composite(
		golang.MapType(golang.Named("Name"), golang.Named("string")),
		elts...,
	))
	return file
}

func writeGo(root *path.Root, name string, file *golang.File) error {
	var buf bytes.Buffer
	if err := file.Make(&buf); err != nil {
		return err
	}
	return path.New(name).WriteFile(root, buf.Bytes(), 0o644)
}
