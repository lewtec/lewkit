package unikernel

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/test"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if handled, err := gotool.Child(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestSentinels(t *testing.T) {
	require.True(t, Firmware("uefi"))
	require.True(t, Firmware("mbr"))
	require.False(t, Firmware("linux"))
	err := Build(t.Context(), Options{GOOS: "uefi", GOARCH: "arm64", Dir: "d", Out: "o"})
	require.ErrorIs(t, err, ErrArch)
	t.Setenv("TAMAGO", "")
	err = Build(t.Context(), Options{GOOS: "mbr", GOARCH: "amd64", Dir: "d", Out: "o"})
	require.ErrorIs(t, err, ErrTAMAGO)
}

func TestRewriteObjectHeader(t *testing.T) {
	raw := []byte("!<arch>\ngo object tamago amd64 go1.27.2 GOAMD64=v1 X:a\n$$\n")
	out, err := rewriteObjectHeaders(raw, "go1.27.0")
	require.NoError(t, err)
	require.Equal(t, "go1.27.0", objectVersion(out))
	require.Contains(t, string(out), "go object tamago amd64 go1.27.0 ")
	_, err = rewriteObjectHeaders(raw, "go1.27.10")
	require.Error(t, err)
}

func TestMBRImageBoots(t *testing.T) {
	if _, err := exec.LookPath("as"); err != nil {
		test.Skip(t, "as is not installed")
	}
	if _, err := exec.LookPath("ld"); err != nil {
		test.Skip(t, "ld is not installed")
	}
	// Apple as and lld reject the GNU ELF emulation this image links with.
	if err := exec.Command("ld", "-m", "elf_x86_64", "--version").Run(); err != nil {
		test.Skip(t, "gnu ld")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "guest.S")
	require.NoError(t, os.WriteFile(src, []byte(guestSrc), 0o644))
	obj := filepath.Join(dir, "guest.o")
	elf := filepath.Join(dir, "guest.elf")
	require.NoError(t, exec.Command("as", "--64", "-o", obj, src).Run())
	require.NoError(t, exec.Command("ld", "-m", "elf_x86_64", "-Ttext=0x10010000", "-e", "_start", "--oformat", "elf64-x86-64", "-o", elf, obj).Run())
	raw, err := os.ReadFile(elf)
	require.NoError(t, err)
	stage, err := assembleBoot(t.Context())
	require.NoError(t, err)
	img, err := packMBR(stage, raw)
	require.NoError(t, err)
	require.Equal(t, []byte{0x55, 0xaa}, img[510:512])
	require.Equal(t, []byte{0x7f, 'E', 'L', 'F'}, img[33*512:33*512+4])

	qemu := os.Getenv("QEMU_SYSTEM")
	if qemu == "" {
		test.Skip(t, "QEMU_SYSTEM is unset")
	}
	disk := filepath.Join(dir, "disk.img")
	require.NoError(t, os.WriteFile(disk, img, 0o644))
	serial := filepath.Join(dir, "serial.txt")
	err = boot(t.Context(), "mbr", disk, serial)
	require.NoError(t, err)
	got, err := os.ReadFile(serial)
	require.NoError(t, err)
	require.Equal(t, "A\n", string(got))
}

func TestGoHello(t *testing.T) {
	if os.Getenv("TAMAGO") == "" || os.Getenv("QEMU_SYSTEM") == "" {
		test.Skip(t, "TAMAGO or QEMU_SYSTEM is unset")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module hello\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello from the unikernel\") }\n"), 0o644))
	modBefore, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)

	for _, goos := range []string{"mbr", "uefi"} {
		t.Run(goos, func(t *testing.T) {
			if goos == "uefi" && (os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "") {
				test.Skip(t, "OVMF_CODE or OVMF_VARS is unset")
			}
			out := filepath.Join(dir, "hello."+goos)
			err := Build(t.Context(), Options{
				Dir:     dir,
				Out:     out,
				Version: "0.1.0",
				AppID:   "br.tec.lew.hello",
				GOOS:    goos,
				GOARCH:  "amd64",
			})
			require.NoError(t, err)
			serial := filepath.Join(dir, "serial-"+goos+".txt")
			require.NoError(t, boot(t.Context(), goos, out, serial))
			got, err := os.ReadFile(serial)
			require.NoError(t, err)
			require.Contains(t, string(got), "hello from the unikernel\n")
		})
	}
	modAfter, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	require.Equal(t, string(modBefore), string(modAfter))
}

func TestExampleConfirms(t *testing.T) {
	if os.Getenv("TAMAGO") == "" || os.Getenv("QEMU_SYSTEM") == "" {
		test.Skip(t, "TAMAGO or QEMU_SYSTEM is unset")
	}
	if _, err := exec.LookPath("as"); err != nil {
		test.Skip(t, "as is not installed")
	}
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	dir := t.TempDir()
	out := filepath.Join(dir, "unikernel.img")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     filepath.Join(root, "examples/unikernel"),
		Out:     out,
		Version: "0.1.0",
		AppID:   "br.tec.lew.eletrocromo.unikernel",
		GOOS:    "mbr",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(dir, "serial.txt")
	sock := filepath.Join(dir, "mon.sock")
	args := []string{
		"-m", "1024",
		"-no-reboot",
		"-display", "none",
		"-monitor", "unix:" + sock + ",server,nowait",
		"-net", "none",
		"-serial", "file:" + serial,
		"-drive", "file=" + out + ",format=raw,if=ide",
		"-boot", "order=c",
	}
	if data := strings.TrimSpace(os.Getenv("QEMU_DATA")); data != "" {
		args = append(args, "-L", data)
	}
	runCtx, runCancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer runCancel()
	cmd := exec.CommandContext(runCtx, os.Getenv("QEMU_SYSTEM"), args...)
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill() }()
	deadline := time.Now().Add(30 * time.Second)
	var got []byte
	for time.Now().Before(deadline) {
		got, _ = os.ReadFile(serial)
		if strings.Contains(string(got), "press enter to confirm\n") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.Contains(t, string(got), "hello from the unikernel\n")
	require.Contains(t, string(got), "press enter to confirm\n")
	conn, err := net.Dial("unix", sock)
	require.NoError(t, err)
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	banner := make([]byte, 256)
	n, _ := conn.Read(banner)
	t.Logf("monitor banner %q", banner[:n])
	_, err = io.WriteString(conn, "sendkey ret\n")
	require.NoError(t, err)
	n, _ = conn.Read(banner)
	t.Logf("monitor reply %q", banner[:n])
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		t.Logf("qemu: %v", err)
	case <-time.After(10 * time.Second):
		got, _ = os.ReadFile(serial)
		require.FailNowf(t, "guest ignored enter", "%s", got)
	}
}

const guestSrc = `
	.code64
	.globl _start
_start:
	movw $0x3f8, %dx
	movb $'A', %al
	outb %al, %dx
	movb $'\n', %al
	outb %al, %dx
	movw $0x604, %dx
	movw $0x2000, %ax
	outw %ax, %dx
1:	hlt
	jmp 1b
`
