// Package sops reads a secret file.
//
// File is a command-line argument. Parse takes a path and Value returns the
// plaintext. A SOPS file encrypted to an age recipient decrypts. A file with
// no SOPS metadata is returned as stored, so a plain PKCS#12 still loads.
//
// A SOPS binary file is a JSON object whose only keys are data and sops.
// Open returns those original bytes unless the path ends in .json, in which
// case the decrypted JSON document is returned. YAML and multi-key JSON come
// back as the decrypted document.
//
// Age identities are read from SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, and
// sops/age/keys.txt under the user config directory. On macOS, XDG_CONFIG_HOME
// is honored first. Cloud KMS, PGP, Vault, SSH keys, and SOPS_AGE_KEY_CMD are
// not used.
package sops
