package androiddex

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/apk"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// miseJavaSpec matches the apk toolchain pin. Run does not install it.
const miseJavaSpec = "java@temurin-17.0.14+7"

const (
	preferredPlatform = "android-35"
	preferredBuild    = "34.0.0"
	dexVersion        = "038"
)

var jdkVersionPattern = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

type javaInputs struct {
	work       string
	classDir   string
	androidJar string
	classpath  []string
	sources    map[string]string
}

func compileJava(ctx context.Context, javac string, in javaInputs) error {
	paths, err := writeSources(ctx, filepath.Join(in.work, "src"), in.sources)
	if err != nil {
		return err
	}
	cp := append([]string{in.androidJar}, in.classpath...)
	args := []string{
		"--release", "8",
		"-encoding", "UTF-8",
		"-d", in.classDir,
		"-classpath", strings.Join(cp, string(os.PathListSeparator)),
	}
	args = append(args, paths...)
	return runTool(ctx, ErrCompile, execdriver.MustCommand(ctx, javac, args...))
}

func writeSources(ctx context.Context, dir string, sources map[string]string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rels := make([]string, 0, len(sources))
	for rel := range sources {
		rels = append(rels, rel)
	}
	slices.Sort(rels)
	paths := make([]string, 0, len(rels))
	for _, rel := range rels {
		if strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
			return nil, fmt.Errorf("%w: %s", ErrPackage, rel)
		}
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(sources[rel]), 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func jarClassTree(ctx context.Context, classDir, dest string) error {
	classes := map[string][]byte{}
	err := filepath.WalkDir(classDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".class") {
			return nil
		}
		rel, err := filepath.Rel(classDir, path)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(filepath.ToSlash(rel), ".class")
		if strings.Contains(name, "..") {
			return fmt.Errorf("%w: %s", ErrCompile, name)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		classes[name] = body
		return nil
	})
	if err != nil {
		return err
	}
	if len(classes) == 0 {
		return fmt.Errorf("%w: no class files", ErrCompile)
	}
	return writeJar(dest, classes)
}

func writeJar(path string, classes map[string][]byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(file)
	names := make([]string, 0, len(classes))
	for name := range classes {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		writer, err := zw.Create(name + ".class")
		if err != nil {
			file.Close()
			return err
		}
		if _, err := writer.Write(classes[name]); err != nil {
			file.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

type d8Inputs struct {
	d8Jar      string
	androidJar string
	outDir     string
	inputs     []string
}

func runD8(ctx context.Context, javaBin string, in d8Inputs) error {
	args := []string{
		"-Xmx2G", "-cp", in.d8Jar, "com.android.tools.r8.D8",
		"--min-api", "26",
		"--lib", in.androidJar,
		"--output", in.outDir,
	}
	args = append(args, in.inputs...)
	if err := runTool(ctx, ErrCompile, execdriver.MustCommand(ctx, javaBin, args...)); err != nil {
		return err
	}
	entries, err := os.ReadDir(in.outDir)
	if err != nil {
		return err
	}
	var dexes []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".dex") {
			dexes = append(dexes, entry.Name())
		}
	}
	if len(dexes) != 1 || dexes[0] != dexName {
		return fmt.Errorf("%w: %s", ErrCompile, strings.Join(dexes, ", "))
	}
	return nil
}

func runTool(ctx context.Context, kind error, cmd *exec.Cmd) error {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := execdriver.Run(ctx, cmd)
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	text := strings.TrimSpace(buf.String())
	if text == "" {
		text = err.Error()
	}
	if len(text) > 4000 {
		text = text[len(text)-4000:]
	}
	return fmt.Errorf("%w: %s", kind, text)
}

func javaTools(ctx context.Context) (string, string, error) {
	if home := strings.TrimSpace(os.Getenv("JAVA_HOME")); home != "" {
		if javacPath, javaPath, ok := jdkBins(ctx, home); ok {
			return javacPath, javaPath, nil
		}
	}
	if bin, err := execdriver.Which(ctx, "javac"); err == nil {
		if javacPath, javaPath, ok := jdkBins(ctx, installRoot(bin)); ok {
			return javacPath, javaPath, nil
		}
	}
	home, err := miseWhere(ctx, miseJavaSpec)
	if err != nil {
		return "", "", fmt.Errorf("%w: javac: %w", ErrTool, err)
	}
	javacPath, javaPath, ok := jdkBins(ctx, home)
	if !ok {
		return "", "", fmt.Errorf("%w: %s", ErrTool, home)
	}
	return javacPath, javaPath, nil
}

func jdkBins(ctx context.Context, home string) (string, string, bool) {
	javacPath := filepath.Join(home, "bin", toolName("javac"))
	javaPath := filepath.Join(home, "bin", toolName("java"))
	if !isFile(javacPath) || !isFile(javaPath) {
		return "", "", false
	}
	text, err := commandText(ctx, javaPath, "-version")
	if err != nil || !jdkVersionOK(text) {
		slog.WarnContext(ctx, "android dex jdk", "home", home, "version", strings.TrimSpace(text), "err", err)
		return "", "", false
	}
	return javacPath, javaPath, true
}

func commandText(ctx context.Context, name string, args ...string) (string, error) {
	cmd := execdriver.MustCommand(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := execdriver.Run(ctx, cmd); err != nil {
		return buf.String(), err
	}
	return buf.String(), nil
}

// jdkVersionOK reports JDK 17 or newer, other than OpenJDK 17.0.2.
// That build throws in CgroupV2Subsystem.
func jdkVersionOK(text string) bool {
	major, minor, patch, ok := jdkVersion(text)
	if !ok || major < 17 {
		return false
	}
	return !(major == 17 && minor == 0 && patch == 2)
}

func jdkVersion(text string) (int, int, int, bool) {
	match := jdkVersionPattern.FindStringSubmatch(text)
	if match == nil {
		return 0, 0, 0, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, 0, false
	}
	minor := 0
	patch := 0
	if match[2] != "" {
		minor, err = strconv.Atoi(match[2])
		if err != nil {
			return 0, 0, 0, false
		}
	}
	if match[3] != "" {
		patch, err = strconv.Atoi(match[3])
		if err != nil {
			return 0, 0, 0, false
		}
	}
	return major, minor, patch, true
}

func installRoot(bin string) string {
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil {
		resolved = bin
	}
	return filepath.Dir(filepath.Dir(resolved))
}

func miseWhere(ctx context.Context, spec string) (string, error) {
	bin, err := execdriver.Which(ctx, "mise")
	if err != nil {
		return "", err
	}
	out, err := commandText(ctx, bin, "where", spec)
	if err != nil {
		return "", err
	}
	found := ""
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "ignoring leftover") {
			continue
		}
		if filepath.IsAbs(line) {
			found = line
		}
	}
	if found == "" {
		return "", fmt.Errorf("%w: %s", ErrTool, spec)
	}
	return found, nil
}

func androidPlatform(ctx context.Context) (string, string, error) {
	sdk, err := apk.AndroidSDK(ctx)
	if err != nil {
		return "", "", err
	}
	jar, err := findAndroidJar(sdk)
	if err != nil {
		return "", "", err
	}
	d8Jar, err := findD8Jar(sdk)
	if err != nil {
		return "", "", err
	}
	return jar, d8Jar, nil
}

func findAndroidJar(sdk string) (string, error) {
	dir := filepath.Join(sdk, "platforms")
	preferred := filepath.Join(dir, preferredPlatform, "android.jar")
	if isFile(preferred) {
		return preferred, nil
	}
	best, err := newestChild(dir, "android.jar")
	if err != nil {
		return "", err
	}
	if best == "" {
		return "", fmt.Errorf("%w: android.jar", ErrTool)
	}
	return filepath.Join(dir, best, "android.jar"), nil
}

func findD8Jar(sdk string) (string, error) {
	dir := filepath.Join(sdk, "build-tools")
	preferred := filepath.Join(dir, preferredBuild, "lib", "d8.jar")
	if isFile(preferred) {
		return preferred, nil
	}
	best, err := newestChild(dir, filepath.Join("lib", "d8.jar"))
	if err != nil {
		return "", err
	}
	if best == "" {
		return "", fmt.Errorf("%w: d8", ErrTool)
	}
	return filepath.Join(dir, best, "lib", "d8.jar"), nil
}

func newestChild(dir, rel string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrTool, dir)
	}
	best := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !isFile(filepath.Join(dir, entry.Name(), rel)) {
			continue
		}
		if best == "" || versionNewer(entry.Name(), best) {
			best = entry.Name()
		}
	}
	return best, nil
}

func versionNewer(a, b string) bool {
	left := versionNumbers(a)
	right := versionNumbers(b)
	n := max(len(left), len(right))
	for i := range n {
		lv, rv := 0, 0
		if i < len(left) {
			lv = left[i]
		}
		if i < len(right) {
			rv = right[i]
		}
		if lv != rv {
			return lv > rv
		}
	}
	return false
}

func versionNumbers(name string) []int {
	var nums []int
	cur := -1
	for _, r := range name {
		if r >= '0' && r <= '9' {
			if cur < 0 {
				cur = 0
			}
			cur = cur*10 + int(r-'0')
			continue
		}
		if cur >= 0 {
			nums = append(nums, cur)
			cur = -1
		}
	}
	if cur >= 0 {
		nums = append(nums, cur)
	}
	return nums
}

func toolName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func checkDex(raw []byte) error {
	if len(raw) < 112 || string(raw[:4]) != "dex\n" {
		return fmt.Errorf("%w: dex magic", ErrCompile)
	}
	if string(raw[4:7]) != dexVersion {
		return fmt.Errorf("%w: dex version %s", ErrCompile, string(raw[4:7]))
	}
	missing, err := unresolvedClasses(raw)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	if len(missing) > 12 {
		missing = missing[:12]
	}
	return fmt.Errorf("%w: %s", ErrCompile, strings.Join(missing, ", "))
}

func unresolvedClasses(raw []byte) ([]string, error) {
	stringIDs := int(u32(raw, 56))
	stringOff := int(u32(raw, 60))
	typeIDs := int(u32(raw, 64))
	typeOff := int(u32(raw, 68))
	classDefs := int(u32(raw, 96))
	classOff := int(u32(raw, 100))
	if classDefs == 0 {
		return nil, fmt.Errorf("%w: dex has no classes", ErrCompile)
	}
	stringsAt := make([]string, stringIDs)
	for i := range stringsAt {
		if stringOff+4*i+4 > len(raw) {
			return nil, fmt.Errorf("%w: dex string", ErrCompile)
		}
		off := int(u32(raw, stringOff+4*i))
		next, err := skipULEB(raw, off)
		if err != nil {
			return nil, err
		}
		end := next
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		if end >= len(raw) {
			return nil, fmt.Errorf("%w: dex string", ErrCompile)
		}
		stringsAt[i] = string(raw[next:end])
	}
	typeName := func(i uint32) (string, error) {
		if int(i) >= typeIDs || typeOff+4*int(i)+4 > len(raw) {
			return "", fmt.Errorf("%w: dex type", ErrCompile)
		}
		idx := u32(raw, typeOff+4*int(i))
		if int(idx) >= len(stringsAt) {
			return "", fmt.Errorf("%w: dex type", ErrCompile)
		}
		return stringsAt[idx], nil
	}
	defined := map[string]bool{}
	for i := range classDefs {
		if classOff+32*i+4 > len(raw) {
			return nil, fmt.Errorf("%w: dex class", ErrCompile)
		}
		desc, err := typeName(u32(raw, classOff+32*i))
		if err != nil {
			return nil, err
		}
		if name, ok := internalName(desc); ok {
			defined[name] = true
		}
	}
	var missing []string
	seen := map[string]bool{}
	for i := range typeIDs {
		desc, err := typeName(uint32(i))
		if err != nil {
			return nil, err
		}
		name, ok := internalName(desc)
		if !ok || defined[name] || frameworkClass(name) || seen[name] {
			continue
		}
		seen[name] = true
		missing = append(missing, name)
	}
	return missing, nil
}

func u32(raw []byte, off int) uint32 {
	return binary.LittleEndian.Uint32(raw[off:])
}

func skipULEB(raw []byte, off int) (int, error) {
	shift := 0
	for {
		if off >= len(raw) || shift > 28 {
			return 0, fmt.Errorf("%w: dex uleb", ErrCompile)
		}
		b := raw[off]
		off++
		if b < 0x80 {
			return off, nil
		}
		shift += 7
	}
}

func internalName(desc string) (string, bool) {
	for strings.HasPrefix(desc, "[") {
		desc = desc[1:]
	}
	if strings.HasPrefix(desc, "L") && strings.HasSuffix(desc, ";") && len(desc) > 2 {
		return desc[1 : len(desc)-1], true
	}
	return "", false
}

func frameworkClass(name string) bool {
	switch {
	case strings.HasPrefix(name, "java/"),
		strings.HasPrefix(name, "javax/"),
		strings.HasPrefix(name, "dalvik/"),
		strings.HasPrefix(name, "sun/"),
		strings.HasPrefix(name, "libcore/"),
		strings.HasPrefix(name, "jdk/"),
		strings.HasPrefix(name, "org/xml"),
		strings.HasPrefix(name, "org/json"),
		strings.HasPrefix(name, "org/w3c"):
		return true
	case strings.HasPrefix(name, "android/") && !strings.HasPrefix(name, "androidx/"):
		return true
	default:
		return false
	}
}
