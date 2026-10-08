package sign

import (
	"errors"
	"fmt"
)

// These are the failures callers tell apart with errors.Is.
var (
	// ErrNotRSA means the PKCS#12 key is not RSA. Apple code-signing
	// certificates and Microsoft Authenticode are issued for RSA keys.
	ErrNotRSA = errors.New("publisher key must be RSA")

	// ErrNoKey means a signature was requested without a private key.
	ErrNoKey = errors.New("publisher key is required")

	// ErrPKCS12 means the PKCS#12 bytes could not be decoded.
	ErrPKCS12 = errors.New("pkcs12")

	// ErrNilContext means the caller did not pass a context.
	ErrNilContext = errors.New("pe: nil context")

	// ErrCertificate means the leaf is missing or was not issued for this key.
	ErrCertificate = errors.New("certificate")

	// ErrPE means the Authenticode signature could not be built or checked.
	ErrPE = errors.New("pe")

	// ErrAPK means the APK signature could not be built or checked.
	ErrAPK = errors.New("apk")

	// ErrCMS means the detached CMS signature could not be built or checked.
	ErrCMS = errors.New("cms")

	// ErrMachO means the Mach-O signature could not be written.
	ErrMachO = errors.New("macho")
)

func cause(kind, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", kind, err)
}

func fail(kind error, msg string) error {
	return fmt.Errorf("%w: %s", kind, msg)
}
