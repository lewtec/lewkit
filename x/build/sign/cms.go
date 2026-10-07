package sign

import (
	"fmt"

	"github.com/smallstep/pkcs7"
)

// SignCMS returns a detached CMS signature over content.
// Linux archives use this. The kernel does not check it; a verifier does.
func (id *Identity) SignCMS(content []byte) ([]byte, error) {
	if err := id.require(); err != nil {
		return nil, err
	}
	sd, err := pkcs7.NewSignedData(content)
	if err != nil {
		return nil, err
	}
	parents := id.Certs[1:]
	if err := sd.AddSignerChain(id.Certs[0], id.Key, parents, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("cms: %w", err)
	}
	sd.Detach()
	der, err := sd.Finish()
	if err != nil {
		return nil, fmt.Errorf("cms: %w", err)
	}
	return der, nil
}

// VerifyCMS checks a detached signature from [Identity.SignCMS].
// It checks the RSA signature. It does not require a public trust anchor.
func VerifyCMS(content, signature []byte) error {
	p7, err := pkcs7.Parse(signature)
	if err != nil {
		return fmt.Errorf("cms: %w", err)
	}
	p7.Content = content
	if err := p7.Verify(); err != nil {
		return fmt.Errorf("cms: %w", err)
	}
	return nil
}
