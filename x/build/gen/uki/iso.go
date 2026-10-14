package uki

import (
	"errors"
	"fmt"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/fs/compose"
	isofs "github.com/lewtec/lewkit/x/fs/iso"
	lewpath "github.com/lewtec/lewkit/x/path"
)

// writeISO packs the UKI as a UEFI El Torito CD.
// The tree is one ref, EFI/BOOT/BOOT*.EFI, composed over the PE bytes.
// The writer copies that filesystem. It does not special-case the boot path.
func writeISO(goarch string, uki []byte) ([]byte, error) {
	if len(uki) == 0 {
		return nil, errors.New("uki: empty image")
	}
	boot, err := efiBootName(goarch)
	if err != nil {
		return nil, err
	}
	base := fstest.MapFS{
		"guest.efi": {Data: uki, Mode: 0o644},
	}
	tree := compose.New()
	err = tree.Add(lewpath.New("EFI/BOOT/"+boot), compose.File{
		Type: compose.TypeRef,
		Mode: 0o644,
		Values: map[string]compose.Slot{
			"guest": compose.Ref("guest.efi"),
		},
	})
	if err != nil {
		return nil, err
	}
	fsys, err := tree.FS(base)
	if err != nil {
		return nil, err
	}
	return isofs.Write(fsys)
}

func efiBootName(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "BOOTX64.EFI", nil
	case "arm64":
		return "BOOTAA64.EFI", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrArch, goarch)
	}
}

func isoBootArch(image []byte) (string, error) {
	if _, err := isofs.ReadFile(image, "EFI/BOOT/BOOTX64.EFI"); err == nil {
		return "amd64", nil
	}
	if _, err := isofs.ReadFile(image, "EFI/BOOT/BOOTAA64.EFI"); err == nil {
		return "arm64", nil
	}
	return "", errors.New("uki: iso has no EFI boot file")
}
