package androiddex

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	residsName = "resids.go"
	javaDir    = "template/app/src/main/java/lewkit"
	javagenDir = "template/app/src/main/javagen"
	hostPkg    = "lewkit"
)

// resourceField is the Java R member for one resids.go constant.
// The packed layout and the host sources use these names.
type resourceField struct {
	class string
	name  string
}

var resourceFields = map[string]resourceField{
	"resIDSplash":           {"id", "splash"},
	"resIDSplashDetail":     {"id", "splash_detail"},
	"resIDSplashLogo":       {"id", "splash_logo"},
	"resIDSplashProgress":   {"id", "splash_progress"},
	"resIDSplashRetry":      {"id", "splash_retry"},
	"resIDSplashStatus":     {"id", "splash_status"},
	"resIDLayoutMain":       {"layout", "activity_main"},
	"resIDIcon":             {"mipmap", "ic_launcher"},
	"resIDAppName":          {"string", "app_name"},
	"resIDLoading":          {"string", "loading"},
	"resIDRetry":            {"string", "retry"},
	"resIDServerFailed":     {"string", "server_failed"},
	"resIDWebviewError":     {"string", "webview_error"},
	"resIDWebviewHTTPError": {"string", "webview_http_error"},
	"resIDTheme":            {"style", "Theme_Eletrocromo"},
	"resIDFilePaths":        {"xml", "file_paths"},
	"resIDNetSec":           {"xml", "network_security_config"},
}

type javaMember struct {
	name string
	line string
}

// hostSources reads the lewkit Java sources and the resource class.
func hostSources(ctx context.Context, dir string) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resids := filepath.Join(dir, residsName)
	if _, err := os.Stat(resids); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPackage, dir)
	}
	ids, err := resourceIDs(ctx, resids)
	if err != nil {
		return nil, err
	}
	resource, err := lewkitResourceJava(ids)
	if err != nil {
		return nil, err
	}
	sources := map[string]string{"lewkit/R.java": resource}
	if err := readJavaDir(ctx, filepath.Join(dir, filepath.FromSlash(javaDir)), sources); err != nil {
		return nil, err
	}
	if err := readJavaGen(ctx, filepath.Join(dir, filepath.FromSlash(javagenDir)), sources); err != nil {
		return nil, err
	}
	if _, ok := sources["lewkit/HostActivity.java"]; !ok {
		return nil, fmt.Errorf("%w: HostActivity.java", ErrPackage)
	}
	if _, ok := sources["lewkit/MainActivity.java"]; !ok {
		return nil, fmt.Errorf("%w: MainActivity.java", ErrPackage)
	}
	return sources, nil
}

func readJavaDir(ctx context.Context, dir string, sources map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrPackage, dir)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			return fmt.Errorf("%w: %s", ErrPackage, name)
		}
		if !strings.HasSuffix(name, ".java") || name == "R.java" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		sources["lewkit/"+name] = string(body)
	}
	return nil
}

func readJavaGen(ctx context.Context, dir string, sources map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrPackage, dir)
	}
	found := false
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			return fmt.Errorf("%w: %s", ErrPackage, name)
		}
		if !strings.HasSuffix(name, ".java.tmpl") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		rendered, err := renderHostTemplate(string(body))
		if err != nil {
			return err
		}
		base := strings.TrimSuffix(name, ".tmpl")
		sources["lewkit/"+base] = rendered
		found = true
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrPackage, dir)
	}
	return nil
}

// renderHostTemplate compiles a javagen file as package lewkit.
// The template's own package is {{.PackageID}}. Imports of lewkit are
// dropped because the class is now in that package.
func renderHostTemplate(body string) (string, error) {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	if len(lines) == 0 || lines[0] != "package {{.PackageID}};" {
		return "", fmt.Errorf("%w: package {{.PackageID}}", ErrPackage)
	}
	out := []string{"package lewkit;"}
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "import lewkit.") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), nil
}

func resourceIDs(ctx context.Context, path string) (map[string]uint32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrResource, path)
	}
	ids := map[string]uint32{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range value.Names {
				if name == nil || !strings.HasPrefix(name.Name, "resID") {
					continue
				}
				if i >= len(value.Values) {
					return nil, fmt.Errorf("%w: %s", ErrResource, name.Name)
				}
				lit, ok := value.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.INT {
					return nil, fmt.Errorf("%w: %s", ErrResource, name.Name)
				}
				number, err := strconv.ParseUint(lit.Value, 0, 32)
				if err != nil {
					return nil, fmt.Errorf("%w: %s", ErrResource, name.Name)
				}
				ids[name.Name] = uint32(number)
			}
		}
	}
	return ids, nil
}

func lewkitResourceJava(ids map[string]uint32) (string, error) {
	groups := map[string][]javaMember{}
	for name, field := range resourceFields {
		value, ok := ids[name]
		if !ok {
			return "", fmt.Errorf("%w: %s", ErrResource, name)
		}
		if err := javaIdent(field.class); err != nil {
			return "", err
		}
		if err := javaIdent(field.name); err != nil {
			return "", err
		}
		groups[field.class] = append(groups[field.class], javaMember{
			name: field.name,
			line: fmt.Sprintf("public static final int %s = 0x%08x;", field.name, value),
		})
	}
	for name := range ids {
		if _, ok := resourceFields[name]; !ok {
			return "", fmt.Errorf("%w: %s", ErrResource, name)
		}
	}
	return renderResourceClass(hostPkg, groups)
}

// resourceJavaFromText renders an aapt R.txt as a Java R class.
func resourceJavaFromText(packageName, text string) (string, error) {
	groups := map[string][]javaMember{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		if err := addResourceLine(groups, line); err != nil {
			return "", err
		}
	}
	return renderResourceClass(packageName, groups)
}

func addResourceLine(groups map[string][]javaMember, line string) error {
	if rest, ok := strings.CutPrefix(line, "int[] "); ok {
		return addResourceArray(groups, line, rest)
	}
	rest, ok := strings.CutPrefix(line, "int ")
	if !ok {
		return fmt.Errorf("%w: %s", ErrResource, line)
	}
	kind, name, value, ok := splitThree(rest)
	if !ok {
		return fmt.Errorf("%w: %s", ErrResource, line)
	}
	if err := javaIdent(kind); err != nil {
		return err
	}
	if err := javaIdent(name); err != nil {
		return err
	}
	if err := javaNumber(value); err != nil {
		return err
	}
	groups[kind] = append(groups[kind], javaMember{
		name: name,
		line: fmt.Sprintf("public static final int %s = %s;", name, value),
	})
	return nil
}

func addResourceArray(groups map[string][]javaMember, line, rest string) error {
	kind, rest, ok := strings.Cut(rest, " ")
	if !ok {
		return fmt.Errorf("%w: %s", ErrResource, line)
	}
	name, body, ok := strings.Cut(rest, " {")
	if !ok || !strings.HasSuffix(body, "}") {
		return fmt.Errorf("%w: %s", ErrResource, line)
	}
	body = strings.TrimSpace(strings.TrimSuffix(body, "}"))
	var values []string
	if body != "" {
		for part := range strings.SplitSeq(body, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if err := javaNumber(part); err != nil {
				return err
			}
			values = append(values, part)
		}
	}
	if err := javaIdent(kind); err != nil {
		return err
	}
	if err := javaIdent(name); err != nil {
		return err
	}
	groups[kind] = append(groups[kind], javaMember{
		name: name,
		line: fmt.Sprintf("public static final int[] %s = { %s };", name, strings.Join(values, ", ")),
	})
	return nil
}

func splitThree(text string) (string, string, string, bool) {
	parts := strings.Fields(text)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func renderResourceClass(packageName string, groups map[string][]javaMember) (string, error) {
	if err := javaPackage(packageName); err != nil {
		return "", err
	}
	var buf strings.Builder
	buf.WriteString("package ")
	buf.WriteString(packageName)
	buf.WriteString(";\n\npublic final class R {\n    private R() {}\n")
	classes := make([]string, 0, len(groups))
	for name := range groups {
		classes = append(classes, name)
	}
	slices.Sort(classes)
	for _, class := range classes {
		members := groups[class]
		slices.SortFunc(members, func(a, b javaMember) int {
			return strings.Compare(a.name, b.name)
		})
		buf.WriteString("\n    public static final class ")
		buf.WriteString(class)
		buf.WriteString(" {\n        private ")
		buf.WriteString(class)
		buf.WriteString("() {}\n")
		for _, member := range members {
			buf.WriteString("        ")
			buf.WriteString(member.line)
			buf.WriteByte('\n')
		}
		buf.WriteString("    }\n")
	}
	buf.WriteString("}\n")
	return buf.String(), nil
}

func javaPackage(name string) error {
	if name == "" {
		return fmt.Errorf("%w: package", ErrResource)
	}
	for part := range strings.SplitSeq(name, ".") {
		if err := javaIdent(part); err != nil {
			return err
		}
	}
	return nil
}

func javaIdent(name string) error {
	if !identPattern.MatchString(name) || javaKeywords[name] {
		return fmt.Errorf("%w: %s", ErrResource, name)
	}
	return nil
}

func javaNumber(value string) error {
	if !numberPattern.MatchString(value) {
		return fmt.Errorf("%w: %s", ErrResource, value)
	}
	return nil
}
