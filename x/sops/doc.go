// Package sops reads a secret file and runs a command with it.
//
// File is a command-line argument. Parse takes a path and Value returns the
// plaintext. A SOPS file encrypted to an age recipient decrypts. A file with
// no SOPS metadata is returned as stored, so a plain PKCS#12 still loads.
//
// A SOPS binary file is a JSON object whose only keys are data and sops.
// Open returns those original bytes unless the path ends in .json, in which
// case the decrypted JSON document is returned. YAML, JSON, and dotenv come
// back as the decrypted document.
//
// Keys is the age search sops uses. SystemKeys reads SOPS_AGE_KEY,
// SOPS_AGE_KEY_FILE, SOPS_AGE_KEY_CMD, an SSH private key, and sops/age/keys.txt.
// On macOS, XDG_CONFIG_HOME is honored first. Env is a dotenv assignment list.
// Target is a tool named kind:name, such as conda:foo. Command runs a program
// with that environment and target.
//
// Age identities are opened with github.com/getsops/sops/v3/age.
package sops
