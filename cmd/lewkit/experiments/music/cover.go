package music

import (
	"encoding/binary"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxTagBytes = 16 << 20

func (lib *Library) trackCover(dir, audioPath string) string {
	if raw := readEmbeddedCover(audioPath); len(raw) > 0 {
		if saved, err := lib.writeCover(audioPath, raw); err == nil {
			return saved
		}
	}
	return folderCover(dir)
}

func (lib *Library) writeCover(audioPath string, raw []byte) (string, error) {
	if lib.covers == "" {
		dir, err := os.MkdirTemp("", "lewkit-music-")
		if err != nil {
			return "", err
		}
		lib.covers = dir
	}
	sum := fnv.New64a()
	sum.Write([]byte(audioPath))
	name := filepath.Join(lib.covers, u64Name(sum.Sum64())+imageExt(raw))
	if err := os.WriteFile(name, raw, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

func u64Name(n uint64) string {
	const digits = "0123456789abcdef"
	var buf [16]byte
	for i := 15; i >= 0; i-- {
		buf[i] = digits[n&0xf]
		n >>= 4
	}
	return string(buf[:])
}

func imageExt(raw []byte) string {
	if len(raw) >= 8 && raw[0] == 0x89 && raw[1] == 'P' && raw[2] == 'N' && raw[3] == 'G' {
		return ".png"
	}
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xd8 {
		return ".jpg"
	}
	return ".img"
}

func folderCover(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	found := map[string]string{}
	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		found[strings.ToLower(name)] = name
		switch strings.ToLower(filepath.Ext(name)) {
		case ".png", ".jpg", ".jpeg":
			images = append(images, name)
		}
	}
	for _, name := range []string{"cover.png", "cover.jpg", "cover.jpeg", "folder.png", "folder.jpg", "front.jpg", "front.png", "album.jpg", "albumart.jpg"} {
		if actual, ok := found[name]; ok {
			return filepath.Join(dir, actual)
		}
	}
	if len(images) == 1 {
		return filepath.Join(dir, images[0])
	}
	return ""
}

func readEmbeddedCover(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	head := make([]byte, 10)
	if _, err := io.ReadFull(file, head); err != nil {
		return nil
	}
	if string(head[:3]) == "ID3" {
		return id3Picture(file, head)
	}
	if string(head[:4]) == "fLaC" {
		return flacPicture(file)
	}
	return nil
}

func id3Picture(file *os.File, head []byte) []byte {
	version := head[3]
	if version < 2 || version > 4 || head[5]&0xC0 != 0 {
		return nil
	}
	size := syncsafe(head[6:10])
	if size < 1 || size > maxTagBytes {
		return nil
	}
	tag := make([]byte, size)
	if _, err := io.ReadFull(file, tag); err != nil {
		return nil
	}
	for len(tag) >= 6 {
		header := 10
		var id string
		var frame int
		if version == 2 {
			header = 6
			id = string(tag[:3])
			frame = int(tag[3])<<16 | int(tag[4])<<8 | int(tag[5])
		} else {
			if len(tag) < 10 {
				return nil
			}
			id = string(tag[:4])
			if version == 4 {
				frame = syncsafe(tag[4:8])
			} else {
				frame = int(binary.BigEndian.Uint32(tag[4:8]))
			}
		}
		if id == "\x00\x00\x00\x00" || id == "\x00\x00\x00" || frame < 0 {
			return nil
		}
		if header+frame > len(tag) {
			return nil
		}
		body := tag[header : header+frame]
		tag = tag[header+frame:]
		if version == 2 && id == "PIC" {
			if image := picImage(body); len(image) > 0 {
				return image
			}
		}
		if id == "APIC" {
			if image := apicImage(body); len(image) > 0 {
				return image
			}
		}
	}
	return nil
}

func apicImage(body []byte) []byte {
	if len(body) < 4 {
		return nil
	}
	encoding := body[0]
	rest := body[1:]
	_, rest = latinField(rest)
	if len(rest) < 1 {
		return nil
	}
	rest = rest[1:]
	_, rest = textField(rest, encoding)
	if len(rest) == 0 {
		return nil
	}
	return rest
}

func picImage(body []byte) []byte {
	if len(body) < 6 {
		return nil
	}
	encoding := body[0]
	rest := body[5:]
	_, rest = textField(rest, encoding)
	if len(rest) == 0 {
		return nil
	}
	return rest
}

func flacPicture(file *os.File) []byte {
	for {
		var head [4]byte
		if _, err := io.ReadFull(file, head[:]); err != nil {
			return nil
		}
		last := head[0]&0x80 != 0
		kind := head[0] & 0x7f
		size := int(head[1])<<16 | int(head[2])<<8 | int(head[3])
		if size < 0 || size > maxTagBytes {
			return nil
		}
		block := make([]byte, size)
		if _, err := io.ReadFull(file, block); err != nil {
			return nil
		}
		if kind == 6 {
			return flacPictureBytes(block)
		}
		if last {
			return nil
		}
	}
}

func flacPictureBytes(block []byte) []byte {
	if len(block) < 32 {
		return nil
	}
	offset := 4
	mime := int(binary.BigEndian.Uint32(block[offset : offset+4]))
	offset += 4 + mime
	if mime < 0 || offset+4 > len(block) {
		return nil
	}
	desc := int(binary.BigEndian.Uint32(block[offset : offset+4]))
	offset += 4 + desc + 16
	if desc < 0 || offset+4 > len(block) {
		return nil
	}
	data := int(binary.BigEndian.Uint32(block[offset : offset+4]))
	offset += 4
	if data < 1 || offset+data > len(block) {
		return nil
	}
	return block[offset : offset+data]
}

func syncsafe(b []byte) int {
	return int(b[0]&0x7f)<<21 | int(b[1]&0x7f)<<14 | int(b[2]&0x7f)<<7 | int(b[3]&0x7f)
}

func latinField(b []byte) (string, []byte) {
	for i, c := range b {
		if c == 0 {
			return string(b[:i]), b[i+1:]
		}
	}
	return "", nil
}

func textField(b []byte, encoding byte) (string, []byte) {
	if encoding == 1 || encoding == 2 {
		for i := 0; i+1 < len(b); i += 2 {
			if b[i] == 0 && b[i+1] == 0 {
				return "", b[i+2:]
			}
		}
		return "", nil
	}
	return latinField(b)
}
