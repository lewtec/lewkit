// Package sops reads a secret file and runs a command with it.
//
// File is a command-line argument. Parse takes a path and Value returns the
// plaintext. A SOPS file decrypts with github.com/getsops/sops/v3/decrypt.
// A file with no SOPS metadata is returned as stored, so a plain PKCS#12
// still loads.
//
// A SOPS binary file is a JSON object whose only keys are data and sops.
// Open returns those original bytes unless the path ends in .json, in which
// case the decrypted JSON document is returned. YAML, JSON, and dotenv come
// back as the decrypted document.
//
// Encrypt runs the sops program. The creation rule comes from .sops.yaml
// above the output path. A missing config file is an error.
//
// Env is a dotenv assignment list. Command runs a program with that environment.
package sops
