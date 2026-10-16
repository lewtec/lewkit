// Package uki boots a Linux guest from a ready-made kernel.
//
// linux_uki is one PE: a systemd EFI stub, the kernel, the command line,
// and an initramfs. linux_iso is that PE on an El Torito UEFI CD written
// from a composed filesystem.
// The app is /app. /init loads the kernel modules from that same package
// and execs it. An empty Options.Kernel downloads Alpine linux-virt for GOARCH.
package uki

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/version"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/release"
)

var (
	ErrQEMU            = errs.Wrap(errs.ErrNotFound, "uki: qemu not found")
	ErrOVMF            = errors.New("uki: OVMF_CODE or OVMF_VARS is unset")
	errApkNoKernel     = errs.Wrap(errs.ErrNotFound, "uki: apk has no vmlinuz-virt")
	errApkNoModules    = errs.Wrap(errs.ErrNotFound, "uki: apk has no modules.dep")
	errEmptyOutput     = errs.Wrap(errs.ErrEmpty, "uki: empty output")
	errImageNotPEOrISO = errs.Wrap(errs.ErrInvalid, "uki: image is not a PE or an ISO")
	// errKept adds no words. The other values are the sentence.
	errKept = errors.New("")
	errUKI  = errors.New("uki")
)

// guestCmdline is stored in the UKI. QEMU does not pass -append.
// tty0 is the screen. ttyS0 is last, so it stays /dev/console.
const guestCmdline = "console=tty0 console=ttyS0,115200 panic=1 rdinit=/init"

// Guest reports whether goos is a Linux guest image.
// linux_uki is the PE. linux_iso is the bootable CD.
func Guest(goos string) bool {
	return goos == "linux_uki" || goos == "linux_iso"
}

// Options is one guest image.
// Kernel is a ready-made vmlinuz. Empty uses the pinned Alpine linux-virt
// package for GOARCH. That package supplies the modules either way.
// GOOS linux_iso writes a CD. Empty and linux_uki write the PE.
type Options struct {
	Dir     string
	Out     string
	AppID   string
	Version string
	GOOS    string
	GOARCH  string
	Kernel  string
}

// Build compiles Dir and writes one UKI at Out.
func Build(ctx context.Context, opt Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opt.GOARCH == "" {
		opt.GOARCH = runtime.GOARCH
	}
	if _, err := AlpineArch(opt.GOARCH); err != nil {
		return err
	}
	switch opt.GOOS {
	case "", "linux_uki", "linux_iso":
	default:
		return fmt.Errorf("%w: goos %s", errUKI, opt.GOOS)
	}
	if strings.TrimSpace(opt.Out) == "" {
		return errEmptyOutput
	}
	// The pinned package is the kernel and the modules. A caller-supplied
	// vmlinuz replaces only the image; the modules stay this package.
	apk, err := fetchKernel(ctx, opt.GOARCH)
	if err != nil {
		return err
	}
	image, err := readKernel(apk, true)
	if err != nil {
		return err
	}
	if supplied := strings.TrimSpace(opt.Kernel); supplied != "" {
		raw, err := os.ReadFile(supplied)
		if err != nil {
			return err
		}
		image.kernel = raw
	}
	work, err := os.MkdirTemp("", release.Name()+"-uki-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	appPath := filepath.Join(work, "app")
	initPath := filepath.Join(work, "init")
	if err := compile(ctx, opt, appPath, initPath); err != nil {
		return err
	}
	app, err := os.ReadFile(appPath)
	if err != nil {
		return err
	}
	initBin, err := os.ReadFile(initPath)
	if err != nil {
		return err
	}
	libs, err := dynamicLibs(ctx, opt.GOARCH, appPath)
	if err != nil {
		return err
	}
	blob, err := pack(image, initBin, app, libs)
	if err != nil {
		return err
	}
	stub, err := fetchStub(ctx, opt.GOARCH)
	if err != nil {
		return err
	}
	ukiBytes, err := assembleUKI(stub, image.kernel, blob, []byte(guestCmdline), []byte(image.version), osRelease(opt.Version))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opt.Out), 0o755); err != nil {
		return err
	}
	if opt.GOOS == "linux_iso" {
		iso, err := writeISO(opt.GOARCH, ukiBytes)
		if err != nil {
			return err
		}
		return os.WriteFile(opt.Out, iso, 0o644)
	}
	return os.WriteFile(opt.Out, ukiBytes, 0o644)
}

func osRelease(version string) []byte {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "0"
	}
	return []byte("NAME=lewkit\nID=lewkit\nVERSION_ID=" + version + "\n")
}

type kernelImage struct {
	version string
	kernel  []byte
	files   map[string][]byte
	order   []string
}

// readKernel reads vmlinuz and, when modules is set, the module closure.
// path is an apk when modules is set, and a raw vmlinuz otherwise.
func readKernel(file string, modules bool) (kernelImage, error) {
	if !modules {
		raw, err := os.ReadFile(file)
		if err != nil {
			return kernelImage{}, err
		}
		return kernelImage{kernel: raw}, nil
	}
	f, err := os.Open(file)
	if err != nil {
		return kernelImage{}, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return kernelImage{}, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var image kernelImage
	var dep, builtin string
	staged := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return kernelImage{}, err
		}
		name := pathClean(hdr.Name)
		switch {
		case name == "boot/vmlinuz-virt" || strings.HasSuffix(name, "/vmlinuz-virt"):
			raw, err := io.ReadAll(tr)
			if err != nil {
				return kernelImage{}, err
			}
			image.kernel = raw
		case strings.HasSuffix(name, "/modules.dep"):
			raw, err := io.ReadAll(tr)
			if err != nil {
				return kernelImage{}, err
			}
			dep = string(raw)
			image.version = path.Base(path.Dir(name))
		case strings.HasSuffix(name, "/modules.builtin"):
			raw, err := io.ReadAll(tr)
			if err != nil {
				return kernelImage{}, err
			}
			builtin = string(raw)
		case strings.Contains(name, "/modules/") && strings.HasSuffix(name, ".ko.gz"):
			raw, err := io.ReadAll(tr)
			if err != nil {
				return kernelImage{}, err
			}
			rel := name[strings.Index(name, "/modules/")+len("/modules/"):]
			if i := strings.IndexByte(rel, '/'); i >= 0 {
				rel = rel[i+1:]
			}
			staged[rel] = raw
		}
	}
	if len(image.kernel) == 0 {
		return kernelImage{}, errApkNoKernel
	}
	if dep == "" {
		return kernelImage{}, errApkNoModules
	}
	order, err := moduleOrder(dep, builtin)
	if err != nil {
		return kernelImage{}, err
	}
	image.order = order
	image.files = map[string][]byte{}
	for _, name := range order {
		raw, ok := staged[name]
		if !ok {
			return kernelImage{}, fmt.Errorf("%w: missing %s", errUKI, name)
		}
		image.files[name] = raw
	}
	return image, nil
}

func pathClean(name string) string {
	name = strings.TrimPrefix(name, "./")
	return path.Clean(name)
}

func compile(ctx context.Context, opt Options, appPath, initPath string) error {
	stamp := version.Info{Version: opt.Version, BuiltBy: release.Name()}.WithAppID(opt.AppID)
	err := gocmd.Command{
		Dir:  opt.Dir,
		Env:  linuxEnv(opt.GOARCH),
		Args: []string{"-trimpath", "-ldflags", stamp, "-o", appPath, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("%w: app: %w", errUKI, err)
	}
	err = gocmd.Command{
		Dir:  guestSource(),
		Env:  linuxEnv(opt.GOARCH),
		Args: []string{"-trimpath", "-o", initPath, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("%w: init: %w", errUKI, err)
	}
	return nil
}

func guestSource() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "guest")
}

// dynamicLibs copies the loader and shared libraries a dynamic app needs.
// Names come from PT_INTERP and DT_NEEDED, and the files come from the
// Alpine 3.24 rootfs packages pinned next to the kernel. purego's fakecgo
// records libc even when cgo is off. A static binary returns nothing.
func dynamicLibs(ctx context.Context, goarch, bin string) ([]cpioFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := elf.Open(bin)
	if err != nil {
		return nil, err
	}
	var interp string
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		raw := make([]byte, prog.Filesz)
		if _, err := prog.ReadAt(raw, 0); err != nil {
			f.Close()
			return nil, err
		}
		interp = strings.TrimRight(string(raw), "\x00")
	}
	needed, needErr := f.DynString(elf.DT_NEEDED)
	f.Close()
	if interp == "" {
		return nil, nil
	}
	if needErr != nil {
		return nil, needErr
	}
	seenName := map[string]bool{}
	seenGuest := map[string]bool{}
	var files []cpioFile
	var sys map[string][]byte
	queue := append([]string{interp}, needed...)
	queue = append(queue, alpineRuntime(goarch)...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if name == "" || seenName[name] {
			continue
		}
		seenName[name] = true
		hits, err := resolveLib(ctx, goarch, name, &sys)
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			return nil, fmt.Errorf("%w: library %s: not found", errUKI, name)
		}
		for _, hit := range hits {
			if seenGuest[hit.guest] {
				continue
			}
			seenGuest[hit.guest] = true
			files = append(files, cpioFile{name: hit.guest, mode: cpioFileMode(cpioTypeFile, 0o755), data: hit.data})
			more, err := elfNeeded(hit.data)
			if err != nil {
				return nil, err
			}
			queue = append(queue, more...)
		}
	}
	if len(files) < 2 {
		return nil, fmt.Errorf("%w: no libraries for %s", errUKI, interp)
	}
	return files, nil
}

type libHit struct {
	guest string
	data  []byte
}

func resolveLib(ctx context.Context, goarch, name string, sys *map[string][]byte) ([]libHit, error) {
	if *sys == nil {
		files, err := fetchAlpineLibs(ctx, goarch)
		if err != nil {
			return nil, err
		}
		*sys = files
	}
	return sysrootHits(*sys, name), nil
}

func isELF(b []byte) bool {
	return len(b) >= 4 && b[0] == 0x7f && b[1] == 'E' && b[2] == 'L' && b[3] == 'F'
}

func elfNeeded(raw []byte) ([]string, error) {
	f, err := elf.NewFile(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	needed, err := f.DynString(elf.DT_NEEDED)
	if err != nil {
		return nil, nil
	}
	return needed, nil
}

func linuxEnv(goarch string) []string {
	out := make([]string, 0, 16)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GOOS=") || strings.HasPrefix(env, "GOARCH=") || strings.HasPrefix(env, "CGO_ENABLED=") {
			continue
		}
		out = append(out, env)
	}
	return append(out, "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+goarch)
}

func pack(image kernelImage, init, app []byte, extra []cpioFile) ([]byte, error) {
	var files []cpioFile
	seen := map[string]bool{}
	add := func(name string, mode uint32, data []byte) {
		ensureDir(&files, seen, path.Dir(name))
		files = append(files, cpioFile{name: name, mode: mode, data: data})
	}
	add("init", cpioFileMode(cpioTypeFile, 0o755), init)
	add("app", cpioFileMode(cpioTypeFile, 0o755), app)
	for _, file := range extra {
		add(file.name, file.mode, file.data)
	}
	base := path.Join("lib/modules", image.version)
	var order bytes.Buffer
	for _, name := range image.order {
		fmt.Fprintf(&order, "%s\n", name)
		add(path.Join(base, name), cpioFileMode(cpioTypeFile, 0o644), image.files[name])
	}
	add(path.Join(base, "load.order"), cpioFileMode(cpioTypeFile, 0o644), order.Bytes())
	var buf bytes.Buffer
	if err := writeCpio(&buf, files); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cpioFileMode(kind, perm uint32) uint32 { return kind | perm }

func ensureDir(files *[]cpioFile, seen map[string]bool, dir string) {
	if dir == "." || dir == "" || seen[dir] {
		return
	}
	ensureDir(files, seen, path.Dir(dir))
	if seen[dir] {
		return
	}
	seen[dir] = true
	*files = append(*files, cpioFile{name: dir, mode: cpioFileMode(cpioTypeDir, 0o755)})
}

// Run boots the UKI under QEMU. Serial is stdout.
func Run(ctx context.Context, image string) error {
	return boot(ctx, image, "", "")
}

// boot starts QEMU. A PE is an EFI application on an ESP. An ISO is a
// CD. shot, when set, is a PPM path polled until the guest has logged a
// kms mode and the framebuffer shows a painted picture.
func boot(ctx context.Context, image, serial, shot string) error {
	return bootArgs(ctx, image, serial, shot, nil)
}

// bootArgs is boot with extra QEMU arguments. An extra "-vga" replaces
// the default "-vga std", so a ramfb or virtio display can be the only screen.
func bootArgs(ctx context.Context, image, serial, shot string, extra []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, goarch, err := bootArch(image)
	if err != nil {
		return err
	}
	bin, err := qemuBin(ctx, goarch)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", release.Name()+"-uki-run-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var drives []string
	switch kind {
	case "iso":
		drives, err = cdromDrives(image, tmp)
	default:
		drives, err = efiDrives(image, goarch, tmp)
	}
	if err != nil {
		return err
	}
	args := []string{
		"-m", "2048",
		"-no-reboot",
		"-display", "none",
		"-net", "none",
	}
	vga := true
	for _, arg := range extra {
		if arg == "-vga" {
			vga = false
		}
	}
	if vga {
		args = append(args, "-vga", "std")
	}
	args = append(args, extra...)
	if goarch == "arm64" {
		args = append([]string{"-machine", "virt", "-cpu", "cortex-a57"}, args...)
	}
	args = append(args, drives...)
	if data := strings.TrimSpace(os.Getenv("QEMU_DATA")); data != "" {
		args = append(args, "-L", data)
	}
	var sock string
	if shot != "" {
		sock = filepath.Join(os.TempDir(), fmt.Sprintf("lewkit-uki-%d.sock", os.Getpid()))
		_ = os.Remove(sock)
		defer os.Remove(sock)
		// QEMU listens. The screendump probe dials that socket.
		args = append(args, "-monitor", "unix:"+sock+",server,nowait")
	}
	if serial == "" {
		args = append(args, "-serial", "stdio")
	} else {
		args = append(args, "-serial", "file:"+serial)
	}
	cmd := execdriver.MustCommand(ctx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() { _ = cmd.Process.Kill() }()
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	if shot == "" {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-wait:
			return err
		}
	}
	conn, err := dialMonitor(ctx, sock, wait, serial)
	if err != nil {
		return err
	}
	defer conn.Close()
	go discardMonitor(conn)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if err := screendump(conn, shot); err != nil {
			return qemuExit(err, serial)
		}
		if guestDrew(serial, shot) {
			return nil
		}
		select {
		case <-ctx.Done():
			return qemuExit(ctx.Err(), serial)
		case err := <-wait:
			return qemuExit(err, serial)
		case <-tick.C:
		}
	}
}

func bootArch(path string) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	hdr := make([]byte, 0x40)
	n, err := f.Read(hdr)
	if n >= 2 && hdr[0] == 'M' && hdr[1] == 'Z' {
		arch, err := archOf(path)
		return "pe", arch, err
	}
	if _, err := f.Seek(0x8000, io.SeekStart); err != nil {
		return "", "", err
	}
	magic := make([]byte, 6)
	if _, err := io.ReadFull(f, magic); err != nil {
		return "", "", errImageNotPEOrISO
	}
	if string(magic[1:]) != "CD001" {
		return "", "", errImageNotPEOrISO
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	arch, err := isoBootArch(raw)
	return "iso", arch, err
}

func archOf(image string) (string, error) {
	f, err := os.Open(image)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, err := f.Read(buf)
	if n == 0 && err != nil {
		return "", err
	}
	return imageArch(buf[:n])
}

func ovmfDrives(tmp string) ([]string, error) {
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
	return []string{
		"-drive", "if=pflash,format=raw,readonly=on,file=" + code,
		"-drive", "if=pflash,format=raw,file=" + varsCopy,
	}, nil
}

func cdromDrives(iso, tmp string) ([]string, error) {
	fw, err := ovmfDrives(tmp)
	if err != nil {
		return nil, err
	}
	return append(fw, "-cdrom", iso), nil
}

func efiDrives(image, goarch, tmp string) ([]string, error) {
	fw, err := ovmfDrives(tmp)
	if err != nil {
		return nil, err
	}
	boot := "BOOTX64.EFI"
	if goarch == "arm64" {
		boot = "BOOTAA64.EFI"
	}
	dir := filepath.Join(tmp, "esp", "EFI", "BOOT")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	in, err := os.Open(image)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	out, err := os.OpenFile(filepath.Join(dir, boot), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return append(fw,
		"-drive", "file=fat:rw:"+filepath.Join(tmp, "esp")+",format=raw,media=disk",
	), nil
}

// guestDrew reports that the guest's kms driver logged a mode and the
// framebuffer is a painted picture. The firmware logo is colored too,
// so the serial line is the gate.
func guestDrew(serial, shot string) bool {
	raw, err := os.ReadFile(serial)
	if err != nil || !strings.Contains(string(raw), "kms:") {
		return false
	}
	return frameDrawn(shot)
}

func dialMonitor(ctx context.Context, sock string, wait <-chan error, serial string) (net.Conn, error) {
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	var last error
	for {
		conn, err := net.DialTimeout("unix", sock, time.Second)
		if err == nil {
			return conn, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, qemuExit(ctx.Err(), serial)
		case err := <-wait:
			return nil, qemuExit(err, serial)
		case <-deadline.C:
			if last == nil {
				last = errors.New("uki: qemu monitor")
			}
			return nil, last
		case <-tick.C:
		}
	}
}

func discardMonitor(conn net.Conn) {
	buf := make([]byte, 4096)
	for {
		if _, err := conn.Read(buf); err != nil {
			return
		}
	}
}

func qemuBin(ctx context.Context, goarch string) (string, error) {
	if goarch == "amd64" {
		if p := strings.TrimSpace(os.Getenv("QEMU_SYSTEM")); p != "" {
			return p, nil
		}
	}
	name := "qemu-system-x86_64"
	if goarch == "arm64" {
		name = "qemu-system-aarch64"
	}
	p, err := execdriver.Which(ctx, name)
	if err != nil {
		return "", ErrQEMU
	}
	return p, nil
}

func screendump(conn net.Conn, shot string) error {
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, "screendump "+shot+"\n"); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	return nil
}

func qemuExit(err error, serial string) error {
	if err == nil {
		err = errors.New("uki: qemu exited")
	}
	tail, readErr := os.ReadFile(serial)
	if readErr != nil || len(tail) == 0 {
		return err
	}
	if len(tail) > 4000 {
		tail = tail[len(tail)-4000:]
	}
	return fmt.Errorf("%w%w\n%s", errKept, err, tail)
}

// frameDrawn reports whether a PPM screendump has a painted picture.
// The VGA text page that QEMU serves before modeset is 720×400 and black.
func frameDrawn(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	hdr, rest, ok := bytes.Cut(raw, []byte("\n"))
	if !ok || string(hdr) != "P6" {
		return false
	}
	dims, rest, ok := bytes.Cut(rest, []byte("\n"))
	if !ok {
		return false
	}
	var w, h int
	if _, err := fmt.Sscanf(string(dims), "%d %d", &w, &h); err != nil || w < 1 || h < 1 {
		return false
	}
	_, pix, ok := bytes.Cut(rest, []byte("\n"))
	if !ok || len(pix) < w*h*3 {
		return false
	}
	bright := 0
	seen := map[[3]byte]struct{}{}
	step := 16
	for y := 0; y < h; y += step {
		for x := 0; x < w; x += step {
			i := (y*w + x) * 3
			r, g, b := pix[i], pix[i+1], pix[i+2]
			if int(r)+int(g)+int(b) > 48 {
				bright++
			}
			seen[[3]byte{r >> 4, g >> 4, b >> 4}] = struct{}{}
		}
	}
	if w == 720 && h == 400 {
		return false
	}
	// A framebuffer console is a few bright glyphs. A painted board fills
	// the window, so the sample has to be dense.
	return bright > 200 && len(seen) >= 2
}
