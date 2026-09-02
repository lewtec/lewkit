package release

import (
	"cmp"
	"fmt"
	"runtime/debug"
	"strings"
)

var version = "dev"

func formatVersion(version, vcs string) (ret string) {
	version = cmp.Or(version, "dev")
	ret = fmt.Sprintf("%s-%s", version, vcs)
	ret = strings.Trim(ret, "-")
	return
}

func Version() string {
	version := strings.TrimSpace(version)
	vcs := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				vcs = setting.Value[:8]
			}
		}
	}
	return formatVersion(version, vcs)
}
