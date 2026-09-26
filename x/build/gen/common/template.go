package common

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

// MaterializeHost prepares outDir, renders src's template/ tree with data,
// and writes eletrocromo.json from configJSON.
func MaterializeHost(src fs.FS, data any, outDir string, force bool, configJSON []byte) error {
	out, err := ResolveCreateOut(outDir, force)
	if err != nil {
		return err
	}
	if err := WalkTemplate(src, data, out); err != nil {
		return err
	}
	return WriteHostJSON(out, configJSON)
}

// WalkTemplate copies src's template/ tree into out, rendering *.tmpl files
// and any other file whose body contains "{{".
func WalkTemplate(src fs.FS, data any, out string) error {
	return WalkTemplateDest(src, data, out, nil)
}

// WalkTemplateDest is WalkTemplate with an optional remapper for destRel
// (Android kotlin sources live under a package-id path).
func WalkTemplateDest(src fs.FS, data any, out string, dest func(rel, destRel string) string) error {
	return fs.WalkDir(src, "template", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel("template", p)
		if err != nil {
			return err
		}
		rel = filepath.FromSlash(rel)
		raw, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		destRel, body, err := RenderFile(rel, raw, data)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if dest != nil {
			destRel = dest(rel, destRel)
		}
		pathOut := filepath.Join(out, destRel)
		if err := os.MkdirAll(filepath.Dir(pathOut), 0o755); err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if strings.HasSuffix(pathOut, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(pathOut, body, mode)
	})
}

// RenderFile copies a static file or executes a template body.
func RenderFile(rel string, raw []byte, data any) (destRel string, body []byte, err error) {
	destRel = rel
	isTmpl := strings.HasSuffix(rel, ".tmpl")
	if isTmpl {
		destRel = strings.TrimSuffix(rel, ".tmpl")
	}
	if !isTmpl && !bytes.Contains(raw, []byte("{{")) {
		return destRel, raw, nil
	}
	name := path.Base(filepath.ToSlash(rel))
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", nil, err
	}
	return destRel, buf.Bytes(), nil
}

// XMLEscape encodes characters that would break plist / XML text nodes.
func XMLEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
