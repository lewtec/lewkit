// Package header writes purego bindings for a C header.
// The tree-sitter C grammar supplies the syntax. Each generated file
// is build-tagged for the C layouts that share it, and dlopens through
// x/ffi/native.
package header

import (
	"bytes"
	"context"
	"fmt"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Config is the lewkit generate header invocation.
type Config struct {
	File    string
	Out     string
	Package string
	ABI     string
	Include []string
	Define  []string
	Libs    []string
	// Bind names the struct whose fields are the C functions. Empty means Symbols.
	// native.Bind dlopens the library and returns that struct.
	Bind string
	// Soft marks every function field optional. A missing symbol stays nil.
	Soft bool
}

// Run parses File once per selected platform and writes Go bindings.
// An empty ABI covers every supported architecture. Platforms whose
// generated source matches share one file and one //go:build line.
// When every platform emits the same Go projection, Run writes one
// file with no build tag. When projections differ, platforms outside
// the matrix receive the linux/amd64 projection under a complement tag.
// Out defaults to <stem>_<abi>.go beside the header. When Out ends in
// .go and only one file is produced, that path is used as given.
// Package is inferred from the output directory when empty.
func Run(ctx context.Context, cfg Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.File == "" {
		return errHeader
	}
	targets, err := Platforms(cfg.ABI)
	if err != nil {
		return err
	}
	pkg := cfg.Package
	if pkg == "" {
		pkg = inferPackage(outputDir(cfg.File, cfg.Out))
	}
	if !token.IsIdentifier(pkg) || token.IsKeyword(pkg) {
		return fmt.Errorf("%w: %s", errPackage, pkg)
	}
	items := make([]rendered, 0, len(targets))
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		unit, err := Parse(ctx, cfg.File, Options{
			Include: cfg.Include,
			Define:  cfg.Define,
			Target:  target,
		})
		if err != nil {
			return fmt.Errorf("%w%s/%s: %w", ErrRun, target.GOOS, target.GOARCH, err)
		}
		unit.BindName = cfg.Bind
		unit.BindSoft = cfg.Soft
		buf, err := Write(unit, pkg, cfg.Libs)
		if err != nil {
			return fmt.Errorf("%w%s/%s: %w", ErrRun, target.GOOS, target.GOARCH, err)
		}
		items = append(items, rendered{target: target, src: buf})
	}
	if cfg.ABI == "" && sameProjection(items) {
		return writeAtomic(bindingPaths(cfg.File, cfg.Out, []string{"abi"})[0], rewriteUniversal(items[0].src))
	}
	groups := groupRendered(items)
	paths := bindingPaths(cfg.File, cfg.Out, assignSlugs(groups))
	for i, group := range groups {
		body := rewriteConstraint(group[0].src, mergeConstraint(targetsOf(group)))
		if err := writeAtomic(paths[i], body); err != nil {
			return err
		}
	}
	if cfg.ABI == "" {
		return writeComplement(cfg.File, cfg.Out, items)
	}
	return nil
}

func sameProjection(items []rendered) bool {
	if len(items) == 0 {
		return false
	}
	key := projectionKey(items[0].src)
	for _, item := range items[1:] {
		if projectionKey(item.src) != key {
			return false
		}
	}
	return true
}

func projectionKey(src []byte) string {
	text := stripBuildLine(string(src))
	var b strings.Builder
	for line := range strings.Lines(text) {
		if strings.HasPrefix(line, "// ABI ") {
			continue
		}
		b.WriteString(line)
	}
	return b.String()
}

func stripBuildLine(text string) string {
	if strings.HasPrefix(text, "//go:build ") {
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			text = text[i+1:]
			if strings.HasPrefix(text, "\n") {
				text = text[1:]
			}
		}
	}
	return text
}

func rewriteUniversal(src []byte) []byte {
	lines := strings.Split(stripBuildLine(string(src)), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "// ABI ") {
			lines[i] = "// Every supported ABI produces this Go projection."
			break
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func writeComplement(file, out string, items []rendered) error {
	var src []byte
	for _, item := range items {
		if item.target.GOOS == "linux" && item.target.GOARCH == "amd64" {
			src = item.src
			break
		}
	}
	if src == nil {
		return nil
	}
	targets := make([]Target, len(items))
	for i := range items {
		targets[i] = items[i].target
	}
	body := rewriteConstraint(src, "!("+mergeConstraint(targets)+")")
	text := string(body)
	const abiNote = "// ABI "
	replaced := false
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, abiNote) {
			lines[i] = "// LP64 projection for platforms outside the ABI matrix."
			replaced = true
			break
		}
	}
	if !replaced {
		return nil
	}
	dir := outputDir(file, out)
	stem := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	if strings.HasSuffix(out, ".go") {
		stem = strings.TrimSuffix(filepath.Base(out), filepath.Ext(out))
	}
	return writeAtomic(filepath.Join(dir, stem+"_other.go"), []byte(strings.Join(lines, "\n")))
}

type rendered struct {
	target Target
	src    []byte
}

func groupRendered(items []rendered) [][]rendered {
	var groups [][]rendered
	index := map[string]int{}
	for _, item := range items {
		key := bodyKey(item.src)
		if n, ok := index[key]; ok {
			groups[n] = append(groups[n], item)
			continue
		}
		index[key] = len(groups)
		groups = append(groups, []rendered{item})
	}
	return groups
}

func bodyKey(src []byte) string {
	text := string(src)
	if strings.HasPrefix(text, "//go:build ") {
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			return text[i+1:]
		}
	}
	return text
}

func rewriteConstraint(src []byte, constraint string) []byte {
	text := string(src)
	if strings.HasPrefix(text, "//go:build ") {
		if i := strings.IndexByte(text, '\n'); i >= 0 {
			return []byte("//go:build " + constraint + text[i:])
		}
	}
	return src
}

func mergeConstraint(targets []Target) string {
	if len(targets) == 1 {
		return targets[0].Constraint()
	}
	arch := targets[0].GOARCH
	osName := targets[0].GOOS
	sameArch := true
	sameOS := true
	var osList, archList []string
	seenOS := map[string]bool{}
	seenArch := map[string]bool{}
	for _, t := range targets {
		if t.GOARCH != arch {
			sameArch = false
		}
		if t.GOOS != osName {
			sameOS = false
		}
		if t.GOOS != "" && !seenOS[t.GOOS] {
			seenOS[t.GOOS] = true
			osList = append(osList, t.GOOS)
		}
		if t.GOARCH != "" && !seenArch[t.GOARCH] {
			seenArch[t.GOARCH] = true
			archList = append(archList, t.GOARCH)
		}
	}
	if sameArch && arch != "" {
		return "(" + strings.Join(osList, " || ") + ") && " + arch
	}
	if sameOS && osName != "" {
		return osName + " && (" + strings.Join(archList, " || ") + ")"
	}
	byArch := map[string][]string{}
	var archOrder []string
	for _, t := range targets {
		if _, ok := byArch[t.GOARCH]; !ok {
			archOrder = append(archOrder, t.GOARCH)
		}
		byArch[t.GOARCH] = append(byArch[t.GOARCH], t.GOOS)
	}
	parts := make([]string, 0, len(archOrder))
	for _, archName := range archOrder {
		oses := byArch[archName]
		if len(oses) == 1 {
			parts = append(parts, "("+oses[0]+" && "+archName+")")
			continue
		}
		parts = append(parts, "(("+strings.Join(oses, " || ")+") && "+archName+")")
	}
	return strings.Join(parts, " || ")
}

func groupSlug(targets []Target) string {
	if len(targets) == 1 {
		return targets[0].GOOS + "_" + targets[0].GOARCH
	}
	id := targets[0].ID
	arch := targets[0].GOARCH
	sameID := true
	sameArch := true
	for _, t := range targets[1:] {
		if t.ID != id {
			sameID = false
		}
		if t.GOARCH != arch {
			sameArch = false
		}
	}
	if sameID && sameArch {
		if id == arch {
			return id
		}
		return id + "_" + arch
	}
	if sameID {
		return id
	}
	return "abi"
}

func assignSlugs(groups [][]rendered) []string {
	slugs := make([]string, len(groups))
	used := map[string]int{}
	for i, group := range groups {
		base := groupSlug(targetsOf(group))
		n := used[base]
		used[base] = n + 1
		if n == 0 {
			slugs[i] = base
			continue
		}
		slugs[i] = fmt.Sprintf("%s_%d", base, n+1)
	}
	return slugs
}

func targetsOf(group []rendered) []Target {
	out := make([]Target, len(group))
	for i := range group {
		out[i] = group[i].target
	}
	return out
}

func outputDir(file, out string) string {
	if out == "" {
		return filepath.Dir(file)
	}
	if strings.HasSuffix(out, ".go") {
		return filepath.Dir(out)
	}
	return out
}

func bindingPaths(file, out string, slugs []string) []string {
	if len(slugs) == 1 && strings.HasSuffix(out, ".go") {
		return []string{out}
	}
	dir := outputDir(file, out)
	stem := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	if strings.HasSuffix(out, ".go") {
		stem = strings.TrimSuffix(filepath.Base(out), filepath.Ext(out))
	}
	paths := make([]string, len(slugs))
	for i, slug := range slugs {
		paths[i] = filepath.Join(dir, stem+"_"+slug+".go")
	}
	return paths
}

func writeAtomic(path string, buf []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func inferPackage(dir string) string {
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			if strings.HasSuffix(name, "_purego.go") || generatedSource(filepath.Join(dir, name)) {
				continue
			}
			pkg, err := packageClause(filepath.Join(dir, name))
			if err == nil && pkg != "" {
				return pkg
			}
		}
	}
	base := filepath.Base(dir)
	if base != "" && base != "." && token.IsIdentifier(base) && !token.IsKeyword(base) {
		r, _ := utf8.DecodeRuneInString(base)
		if unicode.IsLetter(r) {
			return base
		}
	}
	return "bind"
}

func generatedSource(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return bytes.Contains(buf[:n], []byte(generatedBy))
}

func packageClause(file string) (string, error) {
	parsed, err := goparser.ParseFile(token.NewFileSet(), file, nil, goparser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	if parsed.Name == nil {
		return "", errPackage
	}
	return parsed.Name.Name, nil
}
