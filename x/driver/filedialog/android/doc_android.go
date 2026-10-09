//go:build android

package android

import (
	"io/fs"
	"os"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

func init() {
	filedialog.RegisterContent(func(names []string) (fs.FS, error) {
		return FS(names, jniDocuments{})
	})
}

type jniDocuments struct{}

func (jniDocuments) Info(uri string) (Doc, error) {
	text, err := jni.Text(jni.CallStatic("lewkit.Documents", "info", uri))
	if err != nil {
		return Doc{}, err
	}
	rows, err := parseCatalog(text)
	if err != nil {
		return Doc{}, err
	}
	if len(rows) != 1 {
		return Doc{}, errDocumentInfo
	}
	return rows[0], nil
}

func (jniDocuments) List(uri string) ([]Doc, error) {
	text, err := jni.Text(jni.CallStatic("lewkit.Documents", "list", uri))
	if err != nil {
		return nil, err
	}
	return parseCatalog(text)
}

func (jniDocuments) Open(uri string) (fs.File, error) {
	return detach("readFd", uri)
}

func (jniDocuments) Thumb(uri string) (fs.File, error) {
	return detach("thumbFd", uri)
}

func detach(method, uri string) (fs.File, error) {
	fd, err := jni.Int(jni.CallStatic("lewkit.Documents", method, uri))
	if err != nil {
		return nil, err
	}
	if fd < 0 {
		return nil, errDocumentFd
	}
	return os.NewFile(uintptr(fd), uri), nil
}
