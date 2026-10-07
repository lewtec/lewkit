package sops

import (
	"fmt"

	sopsage "github.com/getsops/sops/v3/age"
)

// Stanza is one age recipient and the data key encrypted to it.
type Stanza struct {
	Recipient string
	Enc       string
}

// Keys opens a SOPS data key from age stanzas.
type Keys struct{}

// SystemKeys finds age identities the way sops does.
// It reads SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, an SSH private
// key, and sops/age/keys.txt under the user config directory.
func SystemKeys() Keys { return Keys{} }

// DataKey returns the plaintext data key from the first stanza the identities can open.
func (Keys) DataKey(stanzas []Stanza) ([]byte, error) {
	if len(stanzas) == 0 {
		return nil, ErrNotAge
	}
	var last error
	for _, stanza := range stanzas {
		key := &sopsage.MasterKey{
			Recipient:    stanza.Recipient,
			EncryptedKey: stanza.Enc,
		}
		plain, err := key.Decrypt()
		if err != nil {
			last = err
			continue
		}
		return plain, nil
	}
	if last == nil {
		return nil, ErrNotAge
	}
	return nil, fmt.Errorf("%w: %v", ErrNotAge, last)
}

func dataKey(recipients []ageRecipient) ([]byte, error) {
	stanzas := make([]Stanza, len(recipients))
	for i, rec := range recipients {
		stanzas[i] = Stanza{Recipient: rec.Recipient, Enc: rec.Enc}
	}
	return SystemKeys().DataKey(stanzas)
}
