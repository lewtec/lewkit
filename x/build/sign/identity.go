package sign

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

const rsaBits = 2048

// ErrNotRSA means the PKCS#12 key is not RSA. Apple code-signing certificates
// and Microsoft Authenticode are issued for RSA keys.
var ErrNotRSA = errors.New("publisher key must be RSA")

// ErrNoKey means a signature was requested without a private key.
var ErrNoKey = errors.New("publisher key is required")

// Identity is one RSA publisher key and its certificates, leaf first.
// The same key is presented as a PKCS#12, an APK signer, an Authenticode
// signer, a Mach-O signer, and a detached CMS signer.
type Identity struct {
	Key   *rsa.PrivateKey
	Certs []*x509.Certificate
}

// Generate builds a self-signed RSA-2048 code-signing certificate.
// commonName is the certificate subject. An empty name uses "lewkit".
// The certificate is valid for 25 years because Android update identity
// follows this certificate.
func Generate(commonName string) (*Identity, error) {
	if commonName == "" {
		commonName = "lewkit"
	}
	key, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(25, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &Identity{Key: key, Certs: []*x509.Certificate{cert}}, nil
}

// LoadPKCS12 reads a PKCS#12 that holds an RSA key and a leaf certificate.
// Extra certificates in the file are kept after the leaf.
func LoadPKCS12(data []byte, password string) (*Identity, error) {
	key, cert, cas, err := pkcs12.DecodeChain(data, password)
	if err != nil {
		return nil, fmt.Errorf("pkcs12: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrNotRSA
	}
	if cert == nil {
		return nil, errors.New("pkcs12: missing certificate")
	}
	certs := append([]*x509.Certificate{cert}, cas...)
	return &Identity{Key: rsaKey, Certs: certs}, nil
}

// PKCS12 encodes the key and certificates. password may be empty.
func (id *Identity) PKCS12(password string) ([]byte, error) {
	if err := id.require(); err != nil {
		return nil, err
	}
	return pkcs12.Modern2023.Encode(id.Key, id.Certs[0], id.Certs[1:], password)
}

// CSR is a PKCS#10 request for this key, using the leaf subject.
// Apple issues Developer ID and distribution certificates from this request.
// The issued certificate replaces the leaf via [Identity.WithCertificate].
func (id *Identity) CSR() ([]byte, error) {
	if err := id.require(); err != nil {
		return nil, err
	}
	return x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: id.Certs[0].Subject,
	}, id.Key)
}

// WithCertificate returns a copy that uses cert as the leaf.
// cert must be this identity's public key. Pass an Apple-issued certificate
// here; Android and Windows keep using the self-signed leaf in the original.
func (id *Identity) WithCertificate(cert *x509.Certificate) (*Identity, error) {
	if err := id.require(); err != nil {
		return nil, err
	}
	if cert == nil {
		return nil, errors.New("certificate is required")
	}
	if !id.Key.PublicKey.Equal(cert.PublicKey) {
		return nil, errors.New("certificate was not issued for this key")
	}
	certs := make([]*x509.Certificate, 1, len(id.Certs))
	certs[0] = cert
	if len(id.Certs) > 1 {
		certs = append(certs, id.Certs[1:]...)
	}
	return &Identity{Key: id.Key, Certs: certs}, nil
}

func (id *Identity) require() error {
	if id == nil || id.Key == nil || len(id.Certs) == 0 || id.Certs[0] == nil {
		return ErrNoKey
	}
	return nil
}
