package sign

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"math/big"
	"os"
	"sort"
	"time"

	"github.com/KarpelesLab/authenticode"
)

// PEOptions selects Authenticode attributes. Zero values sign with SHA-256
// and the current time, and skip the RFC 3161 timestamp.
type PEOptions struct {
	ProgramName  string
	TimestampURL string
	SigningTime  time.Time
}

// SignPE returns pe with an RSA Authenticode signature embedded.
// pe is a PE32 or PE32+ image, such as a Go windows .exe.
func (id *Identity) SignPE(ctx context.Context, pe []byte, opts PEOptions) ([]byte, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := id.require(); err != nil {
		return nil, err
	}
	parsed, err := authenticode.Parse(pe)
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	digest := parsed.AuthenticodeDigest(sha256.New())
	spc, err := authenticode.BuildSpcIndirectDataContent(digest, crypto.SHA256)
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	if opts.SigningTime.IsZero() {
		opts.SigningTime = time.Now().UTC()
	}
	cms, err := buildAuthenticodeCMS(ctx, spc, id.Key, id.Certs, opts)
	if err != nil {
		return nil, err
	}
	return parsed.EmbedSignature(cms), nil
}

// SignPEFile signs the Windows executable at path in place.
func (id *Identity) SignPEFile(ctx context.Context, path, programName string) error {
	if ctx == nil {
		return ErrNilContext
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	signed, err := id.SignPE(ctx, raw, PEOptions{ProgramName: programName})
	if err != nil {
		return err
	}
	return os.WriteFile(path, signed, 0o755)
}

// VerifyPE checks that signed embeds this identity's certificate and that
// the Authenticode digest of the image is the digest inside the signature.
func (id *Identity) VerifyPE(signed []byte) error {
	if err := id.require(); err != nil {
		return err
	}
	parsed, err := authenticode.Parse(signed)
	if err != nil {
		return cause(ErrPE, err)
	}
	cms, err := winCert(signed)
	if err != nil {
		return err
	}
	if !bytes.Contains(cms, id.Certs[0].Raw) {
		return fail(ErrPE, "signer certificate is not embedded")
	}
	digest := parsed.AuthenticodeDigest(sha256.New())
	if !bytes.Contains(cms, digest) {
		return fail(ErrPE, "authenticode digest is not in the signature")
	}
	return verifyAuthenticodeRSA(cms, id.Certs[0])
}

func buildAuthenticodeCMS(ctx context.Context, spc []byte, key crypto.Signer, chain []*x509.Certificate, opts PEOptions) ([]byte, error) {
	leaf := chain[0]
	h := crypto.SHA256
	hOID := asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	sigOID := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11} // sha256WithRSAEncryption

	spcValue, err := tlvValue(spc)
	if err != nil {
		return nil, err
	}
	hh := h.New()
	hh.Write(spcValue)
	messageDigest := hh.Sum(nil)

	signedAttrs, err := signedAttributes(messageDigest, opts.SigningTime.UTC(), opts.ProgramName)
	if err != nil {
		return nil, err
	}
	setForSigning := append([]byte{0x31}, signedAttrs[1:]...)
	hh.Reset()
	hh.Write(setForSigning)
	attrDigest := hh.Sum(nil)
	signature, err := key.Sign(rand.Reader, attrDigest, h)
	if err != nil {
		return nil, cause(ErrPE, err)
	}

	var unsigned asn1.RawValue
	if opts.TimestampURL != "" {
		token, err := authenticode.RequestTimestamp(ctx, opts.TimestampURL, signature, h)
		if err != nil {
			return nil, cause(ErrPE, err)
		}
		attrDER, err := asn1.Marshal(attribute{
			Type:   asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 3, 3, 1},
			Values: asn1.RawValue{FullBytes: tlv(0x31, token)},
		})
		if err != nil {
			return nil, cause(ErrPE, err)
		}
		unsigned = asn1.RawValue{FullBytes: tlv(0xA1, attrDER)}
	}

	si := signerInfo{
		Version: 1,
		SID: issuerAndSerial{
			Issuer:       asn1.RawValue{FullBytes: leaf.RawIssuer},
			SerialNumber: leaf.SerialNumber,
		},
		DigestAlgorithm:    algIDNull(hOID),
		SignedAttrs:        asn1.RawValue{FullBytes: signedAttrs},
		SignatureAlgorithm: algIDNull(sigOID),
		Signature:          signature,
		UnsignedAttrs:      unsigned,
	}
	var certBytes []byte
	for _, c := range chain {
		certBytes = append(certBytes, c.Raw...)
	}
	sd := signedData{
		Version:          1,
		DigestAlgorithms: []algorithmIdentifier{algIDNull(hOID)},
		EncapContentInfo: encapsulated{
			EContentType: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 1, 4},
			EContent:     asn1.RawValue{FullBytes: tlv(0xA0, spc)},
		},
		Certificates: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      certBytes,
		},
		SignerInfos: []signerInfo{si},
	}
	sdDER, err := asn1.Marshal(sd)
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	ci := contentInfo{
		ContentType: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2},
		Content:     asn1.RawValue{FullBytes: tlv(0xA0, sdDER)},
	}
	der, err := asn1.Marshal(ci)
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	return der, nil
}

func signedAttributes(messageDigest []byte, signingTime time.Time, programName string) ([]byte, error) {
	ctVal, err := asn1.Marshal(asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 1, 4})
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	mdVal, err := asn1.Marshal(messageDigest)
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	stVal, err := asn1.Marshal(signingTime)
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	individual, err := asn1.Marshal(asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 1, 21})
	if err != nil {
		return nil, cause(ErrPE, err)
	}
	attrs := []attribute{
		{Type: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}, Values: asn1.RawValue{FullBytes: tlv(0x31, ctVal)}},
		{Type: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}, Values: asn1.RawValue{FullBytes: tlv(0x31, mdVal)}},
		{Type: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}, Values: asn1.RawValue{FullBytes: tlv(0x31, stVal)}},
		{Type: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 1, 11}, Values: asn1.RawValue{FullBytes: tlv(0x31, tlv(0x30, individual))}},
	}
	if programName != "" {
		opus, err := opusInfo(programName)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, attribute{
			Type:   asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 2, 1, 12},
			Values: asn1.RawValue{FullBytes: tlv(0x31, opus)},
		})
	}
	encoded := make([][]byte, 0, len(attrs))
	for _, a := range attrs {
		b, err := asn1.Marshal(a)
		if err != nil {
			return nil, cause(ErrPE, err)
		}
		encoded = append(encoded, b)
	}
	sort.Slice(encoded, func(i, j int) bool {
		return bytes.Compare(encoded[i], encoded[j]) < 0
	})
	return tlv(0xA0, bytes.Join(encoded, nil)), nil
}

func opusInfo(programName string) ([]byte, error) {
	runes := []rune(programName)
	bmp := make([]byte, 2*len(runes))
	for i, r := range runes {
		if r > 0xFFFF {
			return nil, fail(ErrPE, "program name is outside the BMP")
		}
		binary.BigEndian.PutUint16(bmp[2*i:], uint16(r))
	}
	return tlv(0x30, tlv(0xA0, tlv(0x80, bmp))), nil
}

func verifyAuthenticodeRSA(cms []byte, cert *x509.Certificate) error {
	var info contentInfo
	if _, err := asn1.Unmarshal(cms, &info); err != nil {
		return cause(ErrPE, err)
	}
	body, err := tlvValue(info.Content.FullBytes)
	if err != nil {
		body = info.Content.Bytes
	}
	var sd signedData
	if _, err := asn1.Unmarshal(body, &sd); err != nil {
		return cause(ErrPE, err)
	}
	if len(sd.SignerInfos) != 1 {
		return fail(ErrPE, "expected one signer")
	}
	si := sd.SignerInfos[0]
	attrs := si.SignedAttrs.FullBytes
	if len(attrs) < 2 || attrs[0] != 0xA0 {
		return fail(ErrPE, "signed attributes are missing")
	}
	signedAttrs := append([]byte{0x31}, attrs[1:]...)
	if err := cert.CheckSignature(x509.SHA256WithRSA, signedAttrs, si.Signature); err != nil {
		return cause(ErrPE, err)
	}
	return nil
}

func winCert(raw []byte) ([]byte, error) {
	if len(raw) < 64 || raw[0] != 'M' || raw[1] != 'Z' {
		return nil, fail(ErrPE, "not a PE")
	}
	lfanew := int(binary.LittleEndian.Uint32(raw[60:64]))
	opt := lfanew + 24
	if opt+2 > len(raw) {
		return nil, fail(ErrPE, "truncated optional header")
	}
	optSize := int(binary.LittleEndian.Uint16(raw[lfanew+20 : lfanew+22]))
	magic := binary.LittleEndian.Uint16(raw[opt : opt+2])
	var dataDir int
	switch magic {
	case 0x10B:
		dataDir = opt + 96
	case 0x20B:
		dataDir = opt + 112
	default:
		return nil, fail(ErrPE, "unknown optional header")
	}
	certOff := dataDir + 4*8
	if certOff+8 > len(raw) || opt+optSize > len(raw) {
		return nil, fail(ErrPE, "certificate table is missing")
	}
	va := binary.LittleEndian.Uint32(raw[certOff : certOff+4])
	size := binary.LittleEndian.Uint32(raw[certOff+4 : certOff+8])
	if va == 0 || size < 8 || int(va)+int(size) > len(raw) {
		return nil, fail(ErrPE, "signature table is missing")
	}
	hdr := raw[va : va+8]
	dwLength := binary.LittleEndian.Uint32(hdr[0:4])
	if binary.LittleEndian.Uint16(hdr[6:8]) != 0x0002 || int(dwLength) < 8 || int(va)+int(dwLength) > len(raw) {
		return nil, fail(ErrPE, "signature table is truncated")
	}
	return raw[va+8 : va+dwLength], nil
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue
}

type encapsulated struct {
	EContentType asn1.ObjectIdentifier
	EContent     asn1.RawValue
}

type signedData struct {
	Version          int
	DigestAlgorithms []algorithmIdentifier `asn1:"set"`
	EncapContentInfo encapsulated
	Certificates     asn1.RawValue `asn1:"tag:0,implicit,optional"`
	SignerInfos      []signerInfo  `asn1:"set"`
}

type issuerAndSerial struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

type signerInfo struct {
	Version            int
	SID                issuerAndSerial
	DigestAlgorithm    algorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"tag:0,implicit,optional"`
	SignatureAlgorithm algorithmIdentifier
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"tag:1,implicit,optional"`
}

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

func algIDNull(oid asn1.ObjectIdentifier) algorithmIdentifier {
	return algorithmIdentifier{
		Algorithm:  oid,
		Parameters: asn1.RawValue{Tag: asn1.TagNull, FullBytes: []byte{0x05, 0x00}},
	}
}

func tlv(tag byte, value []byte) []byte {
	n := len(value)
	switch {
	case n < 0x80:
		out := make([]byte, 0, 2+n)
		return append(append(out, tag, byte(n)), value...)
	case n <= 0xFF:
		out := make([]byte, 0, 3+n)
		return append(append(out, tag, 0x81, byte(n)), value...)
	default:
		out := make([]byte, 0, 4+n)
		return append(append(out, tag, 0x82, byte(n>>8), byte(n)), value...)
	}
}

func tlvValue(der []byte) ([]byte, error) {
	if len(der) < 2 {
		return nil, fail(ErrPE, "truncated TLV")
	}
	off := 2
	l := int(der[1])
	if l&0x80 != 0 {
		nl := l & 0x7F
		if nl == 0 || nl > 4 || 2+nl > len(der) {
			return nil, fail(ErrPE, "bad long-form length")
		}
		l = 0
		for i := 0; i < nl; i++ {
			l = (l << 8) | int(der[2+i])
		}
		off = 2 + nl
	}
	if off+l > len(der) {
		return nil, fail(ErrPE, "truncated TLV body")
	}
	return der[off : off+l], nil
}
