// Package tool installs versioned external programs into a caller-owned directory.
//
// A spec is backend:ref@version. A bare name uses backend registry and version latest.
// [Open] binds the store root. [Store.Ensure] installs that version when the
// directory is missing and returns the absolute path of one binary.
//
// Backends register with [Register]. GitHub Releases, mise, and curated short
// names load from [github.com/lewtec/lewkit/x/tool/prelude].
// The store does not read a lockfile or choose ~/.local/share. The caller does.
package tool
