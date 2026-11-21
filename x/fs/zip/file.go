package zip

import (
	stdzip "archive/zip"
	"io"
	"io/fs"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

func openStored(ra io.ReaderAt, zf *stdzip.File) (fs.File, error) {
	off, err := zf.DataOffset()
	if err != nil {
		return nil, err
	}
	return lewfs.Section(zf.FileInfo(), ra, off, int64(zf.UncompressedSize64)), nil
}
