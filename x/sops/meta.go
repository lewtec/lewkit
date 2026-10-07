package sops

import "fmt"

type ageRecipient struct {
	Recipient string `json:"recipient" yaml:"recipient"`
	Enc       string `json:"enc" yaml:"enc"`
}

type keyGroup struct {
	Age   []ageRecipient `json:"age" yaml:"age"`
	PGP   []struct{}     `json:"pgp" yaml:"pgp"`
	KMS   []struct{}     `json:"kms" yaml:"kms"`
	GCP   []struct{}     `json:"gcp_kms" yaml:"gcp_kms"`
	Azure []struct{}     `json:"azure_kv" yaml:"azure_kv"`
	Vault []struct{}     `json:"hc_vault" yaml:"hc_vault"`
}

type sopsMeta struct {
	ShamirThreshold         int            `json:"shamir_threshold" yaml:"shamir_threshold"`
	Age                     []ageRecipient `json:"age" yaml:"age"`
	PGP                     []struct{}     `json:"pgp" yaml:"pgp"`
	KMS                     []struct{}     `json:"kms" yaml:"kms"`
	GCP                     []struct{}     `json:"gcp_kms" yaml:"gcp_kms"`
	Azure                   []struct{}     `json:"azure_kv" yaml:"azure_kv"`
	Vault                   []struct{}     `json:"hc_vault" yaml:"hc_vault"`
	KeyGroups               []keyGroup     `json:"key_groups" yaml:"key_groups"`
	LastModified            string         `json:"lastmodified" yaml:"lastmodified"`
	MAC                     string         `json:"mac" yaml:"mac"`
	UnencryptedSuffix       string         `json:"unencrypted_suffix" yaml:"unencrypted_suffix"`
	EncryptedSuffix         string         `json:"encrypted_suffix" yaml:"encrypted_suffix"`
	UnencryptedRegex        string         `json:"unencrypted_regex" yaml:"unencrypted_regex"`
	EncryptedRegex          string         `json:"encrypted_regex" yaml:"encrypted_regex"`
	UnencryptedCommentRegex string         `json:"unencrypted_comment_regex" yaml:"unencrypted_comment_regex"`
	EncryptedCommentRegex   string         `json:"encrypted_comment_regex" yaml:"encrypted_comment_regex"`
	MACOnlyEncrypted        bool           `json:"mac_only_encrypted" yaml:"mac_only_encrypted"`
}

type sopsFile struct {
	Sops *sopsMeta `json:"sops" yaml:"sops"`
}

func (m *sopsMeta) isSops() bool {
	if m == nil {
		return false
	}
	if m.MAC != "" || m.LastModified != "" {
		return true
	}
	if len(m.Age) > 0 || len(m.PGP) > 0 || len(m.KMS) > 0 || len(m.GCP) > 0 || len(m.Vault) > 0 || len(m.Azure) > 0 {
		return true
	}
	return len(m.KeyGroups) > 0
}

func (m *sopsMeta) prepare() error {
	if m.UnencryptedCommentRegex != "" || m.EncryptedCommentRegex != "" {
		return fmt.Errorf("sops: comment encryption rules are not supported")
	}
	n := 0
	for _, rule := range []string{m.UnencryptedSuffix, m.EncryptedSuffix, m.UnencryptedRegex, m.EncryptedRegex} {
		if rule != "" {
			n++
		}
	}
	if n > 1 {
		return fmt.Errorf("sops: only one encryption rule is allowed")
	}
	if n == 0 {
		m.UnencryptedSuffix = unencryptedSuffix
	}
	if m.MAC == "" || m.LastModified == "" {
		return fmt.Errorf("sops: metadata is missing mac or lastmodified")
	}
	return nil
}

func (m *sopsMeta) recipients() ([]ageRecipient, error) {
	groups := m.groups()
	if len(groups) == 0 {
		return nil, ErrNotAge
	}
	if len(groups) > 1 {
		return nil, fmt.Errorf("sops: shamir key groups are not supported")
	}
	if len(groups[0]) == 0 {
		return nil, ErrNotAge
	}
	return groups[0], nil
}

func (m *sopsMeta) groups() [][]ageRecipient {
	if len(m.Age)+len(m.PGP)+len(m.KMS)+len(m.GCP)+len(m.Azure)+len(m.Vault) > 0 {
		return [][]ageRecipient{m.Age}
	}
	if len(m.KeyGroups) == 0 {
		return nil
	}
	out := make([][]ageRecipient, len(m.KeyGroups))
	for i, group := range m.KeyGroups {
		out[i] = group.Age
	}
	return out
}
