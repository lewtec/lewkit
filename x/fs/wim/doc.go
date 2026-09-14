//go:build linux || windows

// Package wim presents one WIM image as an [io/fs.FS].
//
// [Open] and [Images] take an [io.Reader]. The reader must also be an
// [io.ReaderAt]. A missing [io.ReaderAt] returns [ErrNeedReadAt].
// Neither call copies or spools the file.
//
// Image is 1-based, same as DISM and the WIM XML index.
//
//	f, err := os.Open("install.wim")
//	img, err := wim.Open(f, 1)
//	b, err := fs.ReadFile(img, "Windows/Fonts/arial.ttf")
//
// Opened files are streams. They do not implement [io.ReaderAt].
// This package builds on linux and windows only.
package wim
