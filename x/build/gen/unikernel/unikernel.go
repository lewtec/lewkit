// Package unikernel builds a freestanding amd64 Go main as firmware.
//
// uefi is a PE32+ application. mbr is a bootable disk. Both wrap one
// ELF. Stock Go has no uefi or mbr port, and these targets are not part
// of the desktop release matrix. The ELF is compiled and linked by
// x/toolchain/go for GOOS=tamago. TAMAGO lists the standard-library
// archives with GOOSPKG pointed at the runtime/goos overlay, so that
// runtime is built against it. cmd/asm is not vendored.
package unikernel

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/build/version"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/release"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/lewtec/lewkit/x/toolchain/load"
	"github.com/lewtec/lewkit/x/toolchain/tamago"
)

//go:embed boot
var bootSrc embed.FS

const (
	// Guest halt writes 0x21 to the isa-debug-exit port. QEMU returns (n<<1)|1.
	qemuGuestOK = (0x21 << 1) | 1
	// modulePath is the GOOSPKG module. The overlay sources live in goos/.
	modulePath = "github.com/lewtec/lewkit/x/unikernel"
	// compilerVersion is the object header version x/toolchain/go writes.
	compilerVersion = "go1.27.0"
)

// Sentinels for a firmware build or run.
var (
	ErrArch              = errors.New("unikernel: amd64 only")
	ErrTAMAGO            = errors.New("unikernel: TAMAGO is unset")
	ErrQEMU              = errs.Wrap(errs.ErrNotFound, "unikernel: qemu-system-x86_64 not found")
	ErrOVMF              = errors.New("unikernel: OVMF_CODE or OVMF_VARS is unset")
	errDirAndOutRequired = errs.Wrap(errs.ErrRequired, "unikernel: dir and out are required")
	// errKept adds no words. The other values are the sentence.
	errKept                  = errors.New("")
	errObjectNoHeader        = errs.Wrap(errs.ErrNotFound, "unikernel: object has no header")
	errTruncatedObjectHeader = errors.New("unikernel: truncated object header")
	errUnikernel             = errors.New("unikernel")
)

// Options is one firmware build.
// Dir is the main package. Out is the artifact path.
type Options struct {
	Dir     string
	Out     string
	AppID   string
	Version string
	GOOS    string
	GOARCH  string
}

// Firmware reports whether goos is a unikernel target.
func Firmware(goos string) bool {
	return goos == "uefi" || goos == "mbr"
}

// Build compiles Dir with the embedded compiler and writes Out.
// TAMAGO lists the GOOS=tamago archives and assembles the overlay.
func Build(ctx context.Context, opt Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !Firmware(opt.GOOS) {
		return fmt.Errorf("%w: %s is not uefi or mbr", errUnikernel, opt.GOOS)
	}
	if opt.GOARCH != "amd64" {
		return fmt.Errorf("%w: %s", ErrArch, opt.GOARCH)
	}
	if strings.TrimSpace(opt.Dir) == "" || strings.TrimSpace(opt.Out) == "" {
		return errDirAndOutRequired
	}
	tamagoBin := strings.TrimSpace(os.Getenv("TAMAGO"))
	if tamagoBin == "" {
		return ErrTAMAGO
	}
	work, err := os.MkdirTemp("", release.Name()+"-unikernel-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	std, err := listArchives(ctx, tamagoBin, opt.Dir, work)
	if err != nil {
		return err
	}
	if _, ok := std["runtime/goos"]; !ok {
		return fmt.Errorf("%w: runtime/goos", load.ErrNoArchive)
	}
	if err := adoptArchives(work, compilerVersion, std); err != nil {
		return err
	}
	elf := filepath.Join(work, "kernel.elf")
	if err := linkModule(ctx, opt, work, std, elf); err != nil {
		return err
	}
	raw, err := os.ReadFile(elf)
	if err != nil {
		return err
	}
	var image []byte
	switch opt.GOOS {
	case "mbr":
		image, err = mbrImage(ctx, raw)
	default:
		image, err = efiImage(ctx, raw)
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opt.Out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(opt.Out, image, 0o644)
}

// Run boots a uefi or mbr artifact under QEMU.
// Serial is the process stdout. Exit is the QEMU status.
func Run(ctx context.Context, goos, image string) error {
	return boot(ctx, goos, image, "stdio")
}

func boot(ctx context.Context, goos, image, serial string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !Firmware(goos) {
		return fmt.Errorf("%w: %s is not uefi or mbr", errUnikernel, goos)
	}
	bin, err := qemuBin(ctx)
	if err != nil {
		return err
	}
	spec := serial
	if serial != "stdio" {
		spec = "file:" + serial
	}
	args := []string{
		"-m", "1024",
		"-no-reboot",
		"-display", "none",
		"-monitor", "none",
		"-net", "none",
		"-serial", spec,
		"-device", "isa-debug-exit,iobase=0xf4,iosize=0x04",
	}
	if data := strings.TrimSpace(os.Getenv("QEMU_DATA")); data != "" {
		args = append(args, "-L", data)
	}
	tmp, err := os.MkdirTemp("", release.Name()+"-unikernel-run-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	switch goos {
	case "mbr":
		args = append(args, "-drive", "file="+image+",format=raw,if=ide", "-boot", "order=c")
	default:
		extra, err := uefiDrives(image, tmp)
		if err != nil {
			return err
		}
		args = append(args, extra...)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := execdriver.MustCommand(context.WithoutCancel(ctx), bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = execdriver.Run(context.WithoutCancel(ctx), cmd)
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == qemuGuestOK {
		return nil
	}
	return err
}

func qemuBin(ctx context.Context) (string, error) {
	if p := strings.TrimSpace(os.Getenv("QEMU_SYSTEM")); p != "" {
		return p, nil
	}
	p, err := execdriver.Which(ctx, "qemu-system-x86_64")
	if err != nil {
		return "", ErrQEMU
	}
	return p, nil
}

func uefiDrives(efi, tmp string) ([]string, error) {
	code := strings.TrimSpace(os.Getenv("OVMF_CODE"))
	vars := strings.TrimSpace(os.Getenv("OVMF_VARS"))
	if code == "" || vars == "" {
		return nil, ErrOVMF
	}
	raw, err := os.ReadFile(vars)
	if err != nil {
		return nil, err
	}
	varsCopy := filepath.Join(tmp, "vars.fd")
	if err := os.WriteFile(varsCopy, raw, 0o644); err != nil {
		return nil, err
	}
	esp := filepath.Join(tmp, "esp", "EFI", "BOOT")
	if err := os.MkdirAll(esp, 0o755); err != nil {
		return nil, err
	}
	app, err := os.ReadFile(efi)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(esp, "BOOTX64.EFI"), app, 0o644); err != nil {
		return nil, err
	}
	return []string{
		"-drive", "if=pflash,format=raw,readonly=on,file=" + code,
		"-drive", "if=pflash,format=raw,file=" + varsCopy,
		"-drive", "file=fat:rw:" + filepath.Join(tmp, "esp") + ",format=raw,media=disk",
	}, nil
}

func listArchives(ctx context.Context, bin, dir, work string) (map[string]string, error) {
	root, err := moduleRoot(dir)
	if err != nil {
		return nil, err
	}
	overlay := filepath.Join(work, "overlay")
	if err := writeOverlay(overlay); err != nil {
		return nil, err
	}
	gowork := filepath.Join(work, "go.work")
	body := "go 1.27.0\n\nuse " + root + "\nuse " + overlay + "\n"
	if err := os.WriteFile(gowork, []byte(body), 0o644); err != nil {
		return nil, err
	}
	env := tamagoEnv()
	env["GOOSPKG"] = modulePath
	env["GOWORK"] = gowork
	args := []string{
		"list", "-buildvcs=false", "-deps", "-export",
		"-f", "{{if and .Export (or .Standard (or .SFiles .SysoFiles))}}{{.ImportPath}} {{.Export}}{{end}}",
		".",
	}
	out, err := runTool(ctx, bin, dir, env, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: go list: %w", errUnikernel, err)
	}
	archives := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		imp, file, ok := strings.Cut(line, " ")
		if !ok {
			return nil, fmt.Errorf("%w: go list line %q", errUnikernel, line)
		}
		archives[imp] = file
	}
	if _, ok := archives["runtime"]; !ok {
		return nil, fmt.Errorf("%w: runtime", load.ErrNoArchive)
	}
	return archives, nil
}

func writeOverlay(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "goos"), 0o755); err != nil {
		return err
	}
	mod := "module " + modulePath + "\n\ngo 1.27.0\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		return err
	}
	return fs.WalkDir(tamago.Overlay(), "goos", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := fs.ReadFile(tamago.Overlay(), path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "goos", filepath.Base(path)), body, 0o644)
	})
}

func moduleRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	start := abs
	for {
		if _, err := os.Stat(filepath.Join(abs, "go.mod")); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("%w: no go.mod above %s", errUnikernel, start)
		}
		abs = parent
	}
}

func linkModule(ctx context.Context, opt Options, work string, std map[string]string, elf string) error {
	g, err := load.LoadTarget(opt.Dir, []string{"."}, nil, "tamago", "amd64")
	if err != nil {
		return err
	}
	trim, err := filepath.Abs(opt.Dir)
	if err != nil {
		return err
	}
	type built struct {
		id      string
		archive string
		pkg     load.Package
	}
	var pkgs []built
	byPath := map[string]string{}
	var mains int
	for _, pkg := range g.Pkgs {
		if pkg.Name == "main" {
			mains++
		}
		if pkg.Asm {
			return fmt.Errorf("%w: %s", gotool.ErrNoAsm, pkg.ImportPath)
		}
		id := pkg.ImportPath
		if pkg.Name == "main" {
			id = "main"
		}
		archive := filepath.Join(work, "pkg", filepath.FromSlash(id)+".a")
		pkgs = append(pkgs, built{id: id, archive: archive, pkg: pkg})
		byPath[pkg.ImportPath] = archive
		if pkg.Name == "main" {
			byPath["main"] = archive
		}
	}
	if mains != 1 {
		return fmt.Errorf("%w: %d main packages", errUnikernel, mains)
	}
	units := make([]gotool.Unit, 0, len(pkgs))
	for _, b := range pkgs {
		cfg := map[string]string{}
		if runtime, ok := std["runtime"]; ok {
			cfg["runtime"] = runtime
		}
		for _, imp := range b.pkg.Imports {
			if imp == "unsafe" {
				continue
			}
			if a, ok := byPath[imp]; ok {
				cfg[imp] = a
				continue
			}
			a, ok := std[imp]
			if !ok {
				return fmt.Errorf("%w: %s", load.ErrNoArchive, imp)
			}
			cfg[imp] = a
		}
		cfgPath := filepath.Join(work, "cfg", filepath.FromSlash(b.id)+".cfg")
		if err := writeImportCfg(cfgPath, cfg); err != nil {
			return err
		}
		var deps []string
		for _, dep := range b.pkg.Deps {
			for _, other := range pkgs {
				if other.pkg.ImportPath == dep {
					deps = append(deps, other.id)
				}
			}
		}
		units = append(units, gotool.Unit{
			ImportPath: b.id,
			Files:      b.pkg.Files,
			Deps:       deps,
			ImportCfg:  cfgPath,
			Lang:       b.pkg.Lang,
			Archive:    b.archive,
			GOOS:       "tamago",
			GOARCH:     "amd64",
			Trimpath:   trim,
		})
	}
	runtime := make([]gotool.Object, 0, len(std))
	paths := make([]string, 0, len(std))
	for path := range std {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		runtime = append(runtime, gotool.Object{
			ImportPath: path,
			Archive:    std[path],
			GOOS:       "tamago",
			GOARCH:     "amd64",
		})
	}
	stamp := version.Info{Version: opt.Version, BuiltBy: release.Name()}
	ld, err := load.SplitFlags(stamp.WithAppID(opt.AppID))
	if err != nil {
		return err
	}
	ld = append(ld, tamago.LinkFlags...)
	return gotool.BuildObjects(ctx, units, gotool.Target{
		GOOS:    "tamago",
		GOARCH:  "amd64",
		Mode:    "exe",
		Runtime: runtime,
		Ldflags: ld,
	}, elf)
}

func objectVersion(data []byte) string {
	const marker = "go object "
	i := bytes.Index(data, []byte(marker))
	if i < 0 {
		return ""
	}
	fields := bytes.Fields(data[i:])
	if len(fields) < 5 {
		return ""
	}
	return string(fields[4])
}

// rewriteObjectHeaders rewrites a same-length header version. TAMAGO's
// objects are a newer patch than this compiler, and the header must match.
func rewriteObjectHeaders(data []byte, want string) ([]byte, error) {
	const marker = "go object "
	if !bytes.Contains(data, []byte(marker)) {
		return nil, errObjectNoHeader
	}
	out := append([]byte(nil), data...)
	for from := 0; ; {
		rel := bytes.Index(out[from:], []byte(marker))
		if rel < 0 {
			return out, nil
		}
		i := from + rel
		lineEnd := bytes.IndexByte(out[i:], '\n')
		if lineEnd < 0 {
			return nil, errTruncatedObjectHeader
		}
		line := out[i : i+lineEnd]
		fields := bytes.Fields(line)
		if len(fields) < 5 {
			return nil, fmt.Errorf("%w: short object header %q", errUnikernel, line)
		}
		got := string(fields[4])
		prefix := fmt.Sprintf("go object %s %s ", fields[2], fields[3])
		if !bytes.HasPrefix(line, []byte(prefix)) {
			return nil, fmt.Errorf("%w: unexpected object header %q", errUnikernel, line)
		}
		if got != want {
			if len(got) != len(want) {
				return nil, fmt.Errorf("%w: object version %s and compiler %s differ in length", errUnikernel, got, want)
			}
			copy(out[i+len(prefix):], want)
		}
		from = i + lineEnd + 1
	}
}

func adoptArchives(work, want string, std map[string]string) error {
	dir := filepath.Join(work, "std")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for path, file := range std {
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		raw, err = rewriteObjectHeaders(raw, want)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", errUnikernel, path, err)
		}
		dst := filepath.Join(dir, strings.ReplaceAll(path, "/", "_")+".a")
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			return err
		}
		std[path] = dst
	}
	return nil
}

func writeImportCfg(path string, files map[string]string) error {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# import config\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "packagefile %s=%s\n", k, files[k])
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func tamagoEnv() map[string]string {
	set := map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        "tamago",
		"GOARCH":      "amd64",
		"GO111MODULE": "on",
		"GOTOOLCHAIN": "local",
	}
	if strings.TrimSpace(os.Getenv("GOCACHE")) == "" {
		set["GOCACHE"] = filepath.Join(os.TempDir(), "lewkit-unikernel-gocache")
	}
	if strings.TrimSpace(os.Getenv("GOMODCACHE")) == "" {
		set["GOMODCACHE"] = filepath.Join(os.TempDir(), "lewkit-unikernel-gomod")
	}
	return set
}

func runTool(ctx context.Context, bin, dir string, set map[string]string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	for _, dir := range []string{set["GOCACHE"], set["GOMODCACHE"]} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	cmd := execdriver.MustCommand(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = mergeEnv(os.Environ(), set)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := execdriver.Run(ctx, cmd); err != nil {
		return "", fmt.Errorf("%w%w\n%s%s", errKept, err, stderr.String(), stdout.String())
	}
	return stdout.String(), nil
}

func mergeEnv(base []string, set map[string]string) []string {
	out := make([]string, 0, len(base)+len(set))
	seen := map[string]bool{}
	for _, e := range base {
		k, _, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if v, replace := set[k]; replace {
			out = append(out, k+"="+v)
			seen[k] = true
			continue
		}
		out = append(out, e)
	}
	for k, v := range set {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}
