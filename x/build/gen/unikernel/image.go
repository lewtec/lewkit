package unikernel

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
		return nil, fmt.Errorf("unikernel: boot sector missing 0x55AA")
	}
	if len(boot) > elfLBA*mbrBytes {
		return nil, fmt.Errorf("unikernel: stage2 is %d bytes", len(boot))
	}
	img := make([]byte, elfLBA*mbrBytes+len(elf)+bouncePad)
	copy(img, boot)
	copy(img[elfLBA*mbrBytes:], elf)
	sectors := uint32(len(img)/mbrBytes - elfLBA)
	binary.LittleEndian.PutUint32(img[0x1be+12:], sectors)
	return img, nil
}

func assembleBoot(ctx context.Context) ([]byte, error) {
	as, err := exec.LookPath("as")
	if err != nil {
		return nil, fmt.Errorf("unikernel: as: %w", err)
	}
	ld, err := exec.LookPath("ld")
	if err != nil {
		return nil, fmt.Errorf("unikernel: ld: %w", err)
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
		return nil, fmt.Errorf("unikernel: as: %w", err)
	}
	if err := run(ctx, dir, ld, "-m", "elf_i386", "-T", "link.ld", "-o", "boot.bin", "boot.o"); err != nil {
		return nil, fmt.Errorf("unikernel: ld: %w", err)
	}
	return os.ReadFile(filepath.Join(dir, "boot.bin"))
}

func efiImage(ctx context.Context, elf []byte) ([]byte, error) {
	objcopy, err := exec.LookPath("objcopy")
	if err != nil {
		return nil, fmt.Errorf("unikernel: objcopy: %w", err)
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
		return nil, fmt.Errorf("unikernel: objcopy: %w", err)
	}
	b, err := os.ReadFile(efiPath)
	if err != nil {
		return nil, err
	}
	if len(b) < 0x40 || binary.LittleEndian.Uint16(b[0:2]) != 0x5A4D {
		return nil, fmt.Errorf("unikernel: objcopy did not write a PE image")
	}
	lfanew := binary.LittleEndian.Uint32(b[0x3c:])
	chars := int(lfanew) + 4 + 18
	if chars+2 > len(b) {
		return nil, fmt.Errorf("unikernel: PE characteristics out of range")
	}
	// Executable, large-address aware. Some firmware rejects the objcopy default.
	binary.LittleEndian.PutUint16(b[chars:], 0x0226)
	return b, nil
}

func run(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", out, err)
	}
	return nil
}
