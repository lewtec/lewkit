package ios

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
	lewpath "github.com/lewtec/lewkit/x/path"
)

// ErrIPA means the path is not an app bundle to pack.
var ErrIPA = errs.Wrap(errs.ErrInvalid, "ipa")

// Pack writes an IPA for the signed bundle at appDir.
// The archive holds Payload/<bundle name>/ and nothing else.
// appDir is the .app directory. ipaPath is the file to create.
func Pack(ctx context.Context, appDir, ipaPath string) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Stat(appDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrIPA, appDir)
	}
	if err := os.MkdirAll(filepath.Dir(ipaPath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(ipaPath), ".ipa-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpName)
		}
	}()
	zw := zip.NewWriter(tmp)
	if err = writeIPA(ctx, zw, appDir); err != nil {
		return errors.Join(err, zw.Close(), tmp.Close())
	}
	if err = zw.Close(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, ipaPath)
}

func writeIPA(ctx context.Context, zw *zip.Writer, appDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	payload := lewpath.New("Payload")
	if err := writeZipDir(zw, payload.String()+"/"); err != nil {
		return err
	}
	bundle := filepath.Base(appDir)
	return filepath.WalkDir(appDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(appDir, path)
		if err != nil {
			return err
		}
		name := payload.Join(bundle).String()
		if rel != "." {
			name = payload.Join(bundle, filepath.ToSlash(rel)).String()
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return writeZipDir(zw, name+"/")
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate
		header.SetMode(zipFileMode(info.Mode(), rel, bundle))
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
}

// zipFileMode keeps a mode the host already stored.
// Windows does not record a Unix exec bit, so the bundle executable is 0755 in the IPA.
func zipFileMode(mode os.FileMode, rel, bundle string) os.FileMode {
	if mode.Perm()&0o111 == 0 && rel == strings.TrimSuffix(bundle, ".app") {
		return 0o755
	}
	return mode
}

func writeZipDir(zw *zip.Writer, name string) error {
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(os.ModeDir | 0o755)
	_, err := zw.CreateHeader(header)
	return err
}
