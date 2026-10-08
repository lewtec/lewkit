package sign

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"

	"github.com/anchore/quill/quill/macho"
	"github.com/anchore/quill/quill/pki"
	"github.com/anchore/quill/quill/pki/load"
	"github.com/anchore/quill/quill/sign"
)

// SignMachO writes an embedded code signature into the Mach-O at path.
// A nil identity is an ad-hoc signature, the pure-Go stand-in for
// codesign --sign -. bundleID is the code directory identifier.
func SignMachO(id *Identity, path, bundleID string) error {
	if bundleID == "" {
		bundleID = filepath.Base(path)
	}
	m, err := macho.NewFile(path)
	if err != nil {
		return cause(ErrMachO, err)
	}
	defer m.Close()

	if m.HasCodeSigningCmd() {
		if err := m.RemoveSigningContent(); err != nil {
			return cause(ErrMachO, err)
		}
	}

	material, err := signingMaterial(id)
	if err != nil {
		return err
	}

	if err := m.AddEmptyCodeSigningCmd(); err != nil {
		return cause(ErrMachO, err)
	}
	superBlobSize, sbBytes, err := sign.GenerateSigningSuperBlob(bundleID, m, material, "", 0)
	if err != nil {
		return cause(ErrMachO, err)
	}
	if err := sign.UpdateSuperBlobOffsetReferences(m, uint64(len(sbBytes))); err != nil {
		return cause(ErrMachO, err)
	}
	_, sbBytes, err = sign.GenerateSigningSuperBlob(bundleID, m, material, "", superBlobSize)
	if err != nil {
		return cause(ErrMachO, err)
	}
	cmd, _, err := m.CodeSigningCmd()
	if err != nil {
		return cause(ErrMachO, err)
	}
	if err := m.Patch(sbBytes, len(sbBytes), uint64(cmd.DataOffset)); err != nil {
		return cause(ErrMachO, err)
	}
	return nil
}

// SignTree signs path when it is a Mach-O, or every Mach-O under path when
// it is a directory. A nil identity ad-hoc signs. This is how a .app is signed
// without codesign.
func SignTree(id *Identity, root, bundleID string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return SignMachO(id, root, bundleID)
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ok, err := isMachO(path)
		if err != nil || !ok {
			return err
		}
		return SignMachO(id, path, bundleID)
	})
}

func signingMaterial(id *Identity) (pki.SigningMaterial, error) {
	if id == nil {
		return pki.SigningMaterial{}, nil
	}
	if err := id.require(); err != nil {
		return pki.SigningMaterial{}, err
	}
	sm, err := pki.NewSigningMaterialFromP12(load.P12Contents{
		PrivateKey:   id.Key,
		Certificate:  id.Certs[0],
		Certificates: id.Certs[1:],
	}, false)
	if err != nil {
		return pki.SigningMaterial{}, cause(ErrMachO, err)
	}
	return *sm, nil
}

func isMachO(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var b [4]byte
	if _, err := io.ReadFull(f, b[:]); err != nil {
		return false, nil
	}
	magic := binary.LittleEndian.Uint32(b[:])
	switch magic {
	case 0xFEEDFACE, 0xFEEDFACF, 0xCEFAEDFE, 0xCFFAEDFE, 0xCAFEBABE, 0xBEBAFECA:
		return true, nil
	default:
		return false, nil
	}
}
