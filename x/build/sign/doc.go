// Package sign turns one RSA publisher key into the signature each host checks.
//
// The key lives in a PKCS#12. Android gets an APK Signature Scheme v2 and v3
// block, Windows gets an Authenticode signature, Apple gets a Mach-O code
// signature, and any other archive gets a detached CMS signature. The work
// is pure Go, so a Linux build does not shell out to apksigner, osslsigncode,
// codesign, or openssl.
package sign
