package android

import (
	"errors"
	"fmt"
	"os"
)

var errNoDataDir = errors.New("package has no data dir")

// aidUserOffset is Android's per-user uid multiplier.
const aidUserOffset = 100000

// ApplicationInfo.UnmarshalParcel in AndroidGoLab/binder ignores the parcel,
// so PackageManager never fills DataDir. The app can already stat its
// per-user data directory.
func dataDirFor(packageName string, uid int, exists func(string) bool) (string, error) {
	if packageName == "" {
		return "", errNoDataDir
	}
	userID := uid / aidUserOffset
	if userID < 0 {
		userID = 0
	}
	candidates := []string{
		fmt.Sprintf("/data/user/%d/%s", userID, packageName),
		"/data/data/" + packageName,
	}
	for _, dir := range candidates {
		if exists(dir) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("%w: %s", errNoDataDir, packageName)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
