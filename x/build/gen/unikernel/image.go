package unikernel

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/release"
)

const (
	mbrBytes  = 512
	elfLBA    = 33
	bouncePad = 32 * 1024
)

func mbrImage(ctx context.Context, elf []byte) ([]byte, error) {
	boot, err := assembleBoot(ctx)
	if err != nil {
		return nil, err
	}
	return packMBR(boot, elf)
}

func packMBR(boot, elf []byte) ([]byte, error) {
	if len(boot) < mbrBytes || binary.LittleEndian.Uint16(boot[510:512]) != 0xAA55 {
		return nil, fmt.Errorf("%w: boot sector missing 0x55AA", errUnikernel)
	}
	if len(boot) > elfLBA*mbrBytes {
		return nil, fmt.Errorf("%w: stage2 is %d bytes", errUnikernel, len(boot))
	}
	img := make([]byte, elfLBA*mbrBytes+len(elf)+bouncePad)
	copy(img, boot)
	copy(img[elfLBA*mbrBytes:], elf)
	sectors := uint32(len(img)/mbrBytes - elfLBA)
	binary.LittleEndian.PutUint32(img[0x1be+12:], sectors)
	return img, nil
}

func assembleBoot(ctx context.Context) ([]byte, error) {
	as, err := execdriver.Which(ctx, "as")
	if err != nil {
		return nil, fmt.Errorf("%w: as: %w", errUnikernel, err)
	}
	ld, err := execdriver.Which(ctx, "ld")
	if err != nil {
		return nil, fmt.Errorf("%w: ld: %w", errUnikernel, err)
	}
	dir, err := os.MkdirTemp("", release.Name()+"-mbr-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	for _, name := range []string{"boot.S", "link.ld"} {
		body, err := bootSrc.ReadFile("boot/" + name)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			return nil, err
		}
	}
	if err := run(ctx, dir, as, "--32", "-o", "boot.o", "boot.S"); err != nil {
		return nil, fmt.Errorf("%w: as: %w", errUnikernel, err)
	}
	if err := run(ctx, dir, ld, "-m", "elf_i386", "-T", "link.ld", "-o", "boot.bin", "boot.o"); err != nil {
		return nil, fmt.Errorf("%w: ld: %w", errUnikernel, err)
	}
	return os.ReadFile(filepath.Join(dir, "boot.bin"))
}

func efiImage(ctx context.Context, elf []byte) ([]byte, error) {
	objcopy, err := execdriver.Which(ctx, "objcopy")
	if err != nil {
		return nil, fmt.Errorf("%w: objcopy: %w", errUnikernel, err)
	}
	dir, err := os.MkdirTemp("", release.Name()+"-efi-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	elfPath := filepath.Join(dir, "kernel.elf")
	efiPath := filepath.Join(dir, "boot.efi")
	if err := os.WriteFile(elfPath, elf, 0o644); err != nil {
		return nil, err
	}
	err = run(ctx, dir, objcopy,
		"--strip-debug",
		"--output-target", "efi-app-x86_64",
		"--subsystem=efi-app",
		"--image-base", "0x10000000",
		"--stack=0x10000",
		elfPath, efiPath,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: objcopy: %w", errUnikernel, err)
	}
	b, err := os.ReadFile(efiPath)
	if err != nil {
		return nil, err
	}
	if len(b) < 0x40 || binary.LittleEndian.Uint16(b[0:2]) != 0x5A4D {
		return nil, fmt.Errorf("%w: objcopy did not write a PE image", errUnikernel)
	}
	lfanew := binary.LittleEndian.Uint32(b[0x3c:])
	chars := int(lfanew) + 4 + 18
	if chars+2 > len(b) {
		return nil, fmt.Errorf("%w: PE characteristics out of range", errUnikernel)
	}
	// Executable, large-address aware. Some firmware rejects the objcopy default.
	binary.LittleEndian.PutUint16(b[chars:], 0x0226)
	return b, nil
}

func run(ctx context.Context, dir, name string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := execdriver.Run(ctx, cmd); err != nil {
		return fmt.Errorf("%w%s: %w", errKept, out.Bytes(), err)
	}
	return nil
}
