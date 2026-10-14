// Package iso writes an El Torito UEFI CD from an [io/fs.FS].
//
// [Write] copies that tree with go-diskfs. The ISO9660 directory is
// [github.com/diskfs/go-diskfs/filesystem/iso9660]. The boot image is a
// FAT16 copy from [github.com/diskfs/go-diskfs/filesystem/fat16].
// The boot catalog points at the FAT image. EFIBOOT.IMG is only the
// boot image.
package iso
