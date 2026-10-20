package sign

import (
	"context"
	"os"
	"path/filepath"
)

// DebugIdentity is the APK signer used when the publisher key is absent.
// The PKCS#12 is created once under the user cache and reused, which is
// what a debug keystore does for upgrades of an installed package.
func DebugIdentity(ctx context.Context) (*Identity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "lewkit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "android-debug.p12")
	if raw, err := os.ReadFile(path); err == nil {
		return LoadPKCS12(raw, "")
	}
	id, err := Generate("Android Debug")
	if err != nil {
		return nil, err
	}
	raw, err := id.PKCS12("")
	if err != nil {
		return nil, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		if raw, readErr := os.ReadFile(path); readErr == nil {
			return LoadPKCS12(raw, "")
		}
		return nil, err
	}
	return id, nil
}
