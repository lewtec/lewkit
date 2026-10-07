package sign

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/agusibrahim/apksig-go/pkg/algo"
	"github.com/agusibrahim/apksig-go/pkg/apkverifier"
	"github.com/agusibrahim/apksig-go/pkg/apkwriter"
	"github.com/agusibrahim/apksig-go/pkg/datasource"
	"github.com/agusibrahim/apksig-go/pkg/signer"
)

// SignAPK returns apk signed with APK Signature Scheme v2 and v3.
// A debug JAR signature in META-INF is removed first, and an existing
// APK signing block is replaced. minSdk 26 installs the v2 signature.
func (id *Identity) SignAPK(apk []byte) ([]byte, error) {
	if err := id.require(); err != nil {
		return nil, err
	}
	stripped, err := stripJARSignature(apk)
	if err != nil {
		return nil, err
	}
	alg, err := algo.PickAlgorithm(id.Key)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	w := &apkwriter.SignedAPKWriter{
		Src: datasource.NewBytes(stripped),
		Signers: []*signer.SignerConfig{{
			PrivateKey: id.Key,
			Certs:      id.Certs,
			Algorithms: []algo.Algorithm{alg},
		}},
		V3MinSdk: 28,
		V3MaxSdk: 0x7fffffff,
		Align:    true,
	}
	if err := w.Write(&out); err != nil {
		return nil, fmt.Errorf("apk: %w", err)
	}
	return out.Bytes(), nil
}

// SignAPKFile signs the APK at path in place.
func (id *Identity) SignAPKFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	signed, err := id.SignAPK(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(path, signed, 0o644)
}

// VerifyAPK reports whether apk carries a v2 or v3 signature from this identity.
func (id *Identity) VerifyAPK(apk []byte) error {
	if err := id.require(); err != nil {
		return err
	}
	res, err := apkverifier.Verify(datasource.NewBytes(apk), 26, 35)
	if err != nil {
		return fmt.Errorf("apk: %w", err)
	}
	if res == nil || !res.Verified || (!res.V2Verified && !res.V3Verified) {
		return fmt.Errorf("apk: signature rejected")
	}
	want := id.Certs[0].Raw
	for _, got := range res.SignerCerts {
		if bytes.Equal(got, want) {
			return nil
		}
	}
	return fmt.Errorf("apk: signer certificate does not match")
}

func stripJARSignature(apk []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(apk), int64(len(apk)))
	if err != nil {
		return nil, fmt.Errorf("apk: %w", err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range r.File {
		if jarSignature(f.Name) {
			continue
		}
		dst, err := w.CreateHeader(&zip.FileHeader{
			Name:     f.Name,
			Method:   f.Method,
			Modified: f.Modified,
		})
		if err != nil {
			return nil, fmt.Errorf("apk: %w", err)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("apk: %w", err)
		}
		_, copyErr := io.Copy(dst, rc)
		rc.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("apk: %w", copyErr)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("apk: %w", err)
	}
	return buf.Bytes(), nil
}

func jarSignature(name string) bool {
	upper := strings.ToUpper(strings.ReplaceAll(name, "\\", "/"))
	const prefix = "META-INF/"
	if !strings.HasPrefix(upper, prefix) {
		return false
	}
	base := upper[len(prefix):]
	if base == "" || strings.Contains(base, "/") {
		return false
	}
	switch {
	case base == "MANIFEST.MF":
		return true
	case strings.HasSuffix(base, ".SF"), strings.HasSuffix(base, ".RSA"), strings.HasSuffix(base, ".DSA"), strings.HasSuffix(base, ".EC"):
		return true
	default:
		return false
	}
}
