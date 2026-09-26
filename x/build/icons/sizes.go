package icons

// LinuxPNGSizes are standard FreeDesktop icon sizes.
var LinuxPNGSizes = []int{16, 24, 32, 48, 64, 128, 256, 512}

// WindowsICOSizes are multi-resolution .ico entries.
var WindowsICOSizes = []int{16, 24, 32, 48, 64, 128, 256}

// MacOSICNSSizes map ICNS type codes to pixel sizes (PNG payload).
var MacOSICNSSizes = []struct {
	Type string
	Size int
}{
	{"icp4", 16},
	{"icp5", 32},
	{"icp6", 64},
	{"ic07", 128},
	{"ic08", 256},
	{"ic09", 512},
	{"ic10", 1024},
}

// AndroidMipmaps: density directory → launcher size (px).
var AndroidMipmaps = []struct {
	Dir  string
	Size int
}{
	{"mipmap-mdpi", 48},
	{"mipmap-hdpi", 72},
	{"mipmap-xhdpi", 96},
	{"mipmap-xxhdpi", 144},
	{"mipmap-xxxhdpi", 192},
}

// WebPNGSizes for common web icons.
var WebPNGSizes = []struct {
	Name string
	Size int
}{
	{"favicon-16.png", 16},
	{"favicon-32.png", 32},
	{"apple-touch-icon.png", 180},
}

// ExpectedRelative lists files that must exist for a "complete" tree (skip regen).
func ExpectedRelative() []string {
	out := []string{
		"manifest.json",
		"source/master.png",
		"windows/icon.ico",
		"macos/icon.icns",
		"web/favicon.ico",
		"web/favicon-16.png",
		"web/favicon-32.png",
		"web/apple-touch-icon.png",
	}
	for _, s := range LinuxPNGSizes {
		out = append(out, "linux/icon-"+itoa(s)+".png")
	}
	for _, m := range AndroidMipmaps {
		out = append(out, "android/"+m.Dir+"/ic_launcher.png")
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
