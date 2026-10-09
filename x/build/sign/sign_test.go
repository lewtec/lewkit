package sign

import (
	"archive/zip"
	"bytes"
	"crypto/x509"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anchore/quill/quill/macho"
	"github.com/stretchr/testify/require"
)

func TestIdentityPKCS12AndCSR(t *testing.T) {
	id, err := Generate("Publisher")
	require.NoError(t, err)
	require.Equal(t, 2048, id.Key.N.BitLen())

	p12, err := id.PKCS12("secret")
	require.NoError(t, err)
	loaded, err := LoadPKCS12(p12, "secret")
	require.NoError(t, err)
	require.True(t, id.Key.Equal(loaded.Key))
	require.True(t, id.Certs[0].Equal(loaded.Certs[0]))

	csrDER, err := id.CSR()
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDER)
	require.NoError(t, err)
	require.Equal(t, "Publisher", csr.Subject.CommonName)
	require.NoError(t, csr.CheckSignature())

	_, err = LoadPKCS12(p12, "nope")
	require.ErrorIs(t, err, ErrPKCS12)

	_, err = id.WithCertificate(nil)
	require.ErrorIs(t, err, ErrCertificate)
	other, err := Generate("Other")
	require.NoError(t, err)
	_, err = id.WithCertificate(other.Certs[0])
	require.ErrorIs(t, err, ErrCertificate)
}

func TestCMSRoundTrip(t *testing.T) {
	id, err := Generate("Publisher")
	require.NoError(t, err)
	body := []byte("linux archive")
	sig, err := id.SignCMS(body)
	require.NoError(t, err)
	require.NoError(t, VerifyCMS(body, sig))
	require.ErrorIs(t, VerifyCMS([]byte("other"), sig), ErrCMS)
}

func TestAPKRoundTrip(t *testing.T) {
	id, err := Generate("Publisher")
	require.NoError(t, err)

	var raw bytes.Buffer
	zw := zip.NewWriter(&raw)
	w, err := zw.Create("AndroidManifest.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte("manifest"))
	require.NoError(t, err)
	w, err = zw.Create("META-INF/CERT.RSA")
	require.NoError(t, err)
	_, err = w.Write([]byte("old-debug-signature"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	signed, err := id.SignAPK(raw.Bytes())
	require.NoError(t, err)
	require.NoError(t, id.VerifyAPK(signed))
	require.NotContains(t, string(signed), "old-debug-signature")
}

func TestSignPENilContext(t *testing.T) {
	id, err := Generate("Publisher")
	require.NoError(t, err)
	_, err = id.SignPE(nil, minimalPE(), PEOptions{})
	require.ErrorIs(t, err, ErrNilContext)
}

func TestPERoundTrip(t *testing.T) {
	id, err := Generate("Publisher")
	require.NoError(t, err)
	signed, err := id.SignPE(t.Context(), minimalPE(), PEOptions{ProgramName: "demo", SigningTime: id.Certs[0].NotBefore})
	require.NoError(t, err)
	require.NoError(t, id.VerifyPE(signed))
	require.ErrorIs(t, id.VerifyPE([]byte("not a pe")), ErrPE)
}

func TestPEGoWindowsExe(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on PATH")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/pe\n\ngo 1.27.0\n"), 0o644))
	exe := filepath.Join(dir, "hello.exe")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	raw, err := os.ReadFile(exe)
	require.NoError(t, err)
	id, err := Generate("Publisher")
	require.NoError(t, err)
	signed, err := id.SignPE(t.Context(), raw, PEOptions{ProgramName: "hello"})
	require.NoError(t, err)
	require.NoError(t, id.VerifyPE(signed))
}

func TestSignGoDarwinBinary(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on PATH")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/macho\n\ngo 1.27.0\n"), 0o644))
	bin := filepath.Join(dir, "hello")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=arm64", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	require.NoError(t, SignMachO(nil, bin, "hello"))
	require.True(t, machoSigned(t, bin))

	id, err := Generate("Publisher")
	require.NoError(t, err)
	require.NoError(t, SignMachO(id, bin, "app.id"))
	require.True(t, machoSigned(t, bin))
}

func TestMachOAdHocAndKeyed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny")
	require.NoError(t, os.WriteFile(path, minimalMachO(), 0o755))
	require.NoError(t, SignMachO(nil, path, "probe.id"))
	require.True(t, machoSigned(t, path))

	keyed := filepath.Join(dir, "keyed")
	require.NoError(t, os.WriteFile(keyed, minimalMachO(), 0o755))
	id, err := Generate("Publisher")
	require.NoError(t, err)
	require.NoError(t, SignMachO(id, keyed, "app.id"))
	require.True(t, machoSigned(t, keyed))

	app := filepath.Join(dir, "Demo.app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(app, 0o755))
	bin := filepath.Join(app, "Demo")
	require.NoError(t, os.WriteFile(bin, minimalMachO(), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Demo.app", "Contents", "Info.plist"), []byte("plist"), 0o644))
	require.NoError(t, SignTree(id, filepath.Join(dir, "Demo.app"), "app.id"))
	require.True(t, machoSigned(t, bin))
}

func machoSigned(t *testing.T, path string) bool {
	t.Helper()
	m, err := macho.NewFile(path)
	require.NoError(t, err)
	defer m.Close()
	return m.HasCodeSigningCmd()
}

func minimalPE() []byte {
	const fileSize = 512
	const optSize = 224
	raw := make([]byte, fileSize)
	raw[0], raw[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(raw[60:64], 64)
	copy(raw[64:], []byte{'P', 'E', 0, 0})
	// COFF: Machine, sections, optional header size.
	binary.LittleEndian.PutUint16(raw[68:70], 0x14c)
	binary.LittleEndian.PutUint16(raw[84:86], optSize)
	opt := 88
	binary.LittleEndian.PutUint16(raw[opt:opt+2], 0x10B)
	binary.LittleEndian.PutUint32(raw[opt+60:opt+64], fileSize)
	binary.LittleEndian.PutUint32(raw[opt+92:opt+96], 16)
	return raw
}

func minimalMachO() []byte {
	const (
		hdr        = 32
		seg        = 72
		ncmd       = 3
		sizeofcmds = ncmd * seg
		filesz     = 4096
	)
	buf := make([]byte, filesz)
	le := binary.LittleEndian
	le.PutUint32(buf[0:], 0xFEEDFACF)
	le.PutUint32(buf[4:], 0x01000007)
	le.PutUint32(buf[8:], 3)
	le.PutUint32(buf[12:], 2)
	le.PutUint32(buf[16:], ncmd)
	le.PutUint32(buf[20:], sizeofcmds)
	le.PutUint32(buf[24:], 1)
	off := hdr
	put := func(name string, vmaddr, vmsize, fileoff, filesize uint64) {
		le.PutUint32(buf[off:], 0x19)
		le.PutUint32(buf[off+4:], seg)
		copy(buf[off+8:], name)
		le.PutUint64(buf[off+24:], vmaddr)
		le.PutUint64(buf[off+32:], vmsize)
		le.PutUint64(buf[off+40:], fileoff)
		le.PutUint64(buf[off+48:], filesize)
		le.PutUint32(buf[off+56:], 7)
		le.PutUint32(buf[off+60:], 5)
		off += seg
	}
	put("__PAGEZERO", 0, 0x100000000, 0, 0)
	put("__TEXT", 0x100000000, 0x1000, 0, filesz)
	put("__LINKEDIT", 0x100001000, 0x1000, filesz, 0)
	return buf
}
