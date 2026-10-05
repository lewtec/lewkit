package vulkan

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// loaderCandidates is the vulkan-1.dll search list.
// systemRoot is %SystemRoot%. extra are absolute paths already resolved
// (system directory, SearchPath). sdkDirs are SDK roots. manifests are
// Khronos registry value names, each a path to an ICD or layer JSON.
// drivers are vulkan-1.dll paths beside those JSON library_path files.
// goarch selects the SDK RunTimeInstaller directory.
func loaderCandidates(systemRoot, goarch string, extra, sdkDirs, manifests, drivers []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	for _, name := range loaderFileNames(goarch) {
		add(name)
		if systemRoot != "" {
			add(filepath.Join(systemRoot, "System32", name))
		}
	}
	for _, path := range extra {
		add(path)
	}
	arch := runtimeDir(goarch)
	for _, sdk := range sdkDirs {
		addSDK(add, sdk, arch)
	}
	for _, manifest := range manifests {
		if manifest == "" {
			continue
		}
		dir := filepath.Dir(manifest)
		for _, name := range loaderFileNames(goarch) {
			add(filepath.Join(dir, name))
			add(filepath.Join(dir, "Bin", name))
		}
		parent := filepath.Dir(dir)
		addSDK(add, parent, arch)
	}
	for _, path := range drivers {
		add(path)
	}
	return out
}

// quotedLibrary returns the first library_path string in an ICD or layer JSON.
func quotedLibrary(text string) string {
	const key = `"library_path"`
	i := strings.Index(text, key)
	if i < 0 {
		return ""
	}
	rest := text[i+len(key):]
	quote := strings.Index(rest, `"`)
	if quote < 0 {
		return ""
	}
	rest = rest[quote+1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return jsonUnescape(rest[:end])
}

func jsonUnescape(text string) string {
	if !strings.Contains(text, `\`) {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); i++ {
		if text[i] != '\\' || i+1 == len(text) {
			b.WriteByte(text[i])
			continue
		}
		switch text[i+1] {
		case '\\', '/':
			b.WriteByte(text[i+1])
			i++
		default:
			b.WriteByte(text[i])
		}
	}
	return b.String()
}

// besideLoader is vulkan-1.dll in the directory of an ICD or layer library.
// library may be absolute or relative to manifest. A relative path is
// resolved against the JSON file's directory.
func besideLoader(manifest, library string) string {
	library = strings.TrimSpace(library)
	if library == "" {
		return ""
	}
	file := library
	if !winAbs(library) {
		dir := winDir(manifest)
		if dir == "" {
			return ""
		}
		rel := strings.TrimPrefix(library, `.\`)
		rel = strings.TrimPrefix(rel, `./`)
		rel = strings.ReplaceAll(rel, `/`, `\`)
		file = dir + `\` + rel
	}
	dir := winDir(file)
	if dir == "" {
		return ""
	}
	return dir + `\vulkan-1.dll`
}

func winBase(path string) string {
	path = strings.ReplaceAll(path, `\`, `/`)
	if i := strings.LastIndex(path, `/`); i >= 0 {
		return path[i+1:]
	}
	return path
}

func winAbs(path string) bool {
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		c := path[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	return strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, `//`)
}

func winDir(path string) string {
	path = strings.TrimRight(path, `\/`)
	i := strings.LastIndexAny(path, `\/`)
	if i <= 0 {
		return ""
	}
	return path[:i]
}

// loaderFileNames are the Vulkan loader filenames for goarch.
// The installed loader is vulkan-1.dll. Driver packages stage a
// different name and copy it to that installed name:
// NVIDIA vulkan-1-x64.dll / vulkan-1-x86.dll / vulkan-1-a64.dll,
// Intel vulkan-1-64.dll / vulkan-1-32.dll, AMD vulkan64.dll / vulkan32.dll.
// Qualcomm stages vulkan-1.dll. The Vulkan runtime installer also writes
// vulkan-1-<major>-<minor>-<patch>-<build>.dll; those are versionedLoaders.
func loaderFileNames(goarch string) []string {
	switch goarch {
	case "386":
		return []string{"vulkan-1.dll", "vulkan-1-x86.dll", "vulkan-1-32.dll", "vulkan32.dll"}
	case "arm64":
		return []string{"vulkan-1.dll", "vulkan-1-a64.dll", "vulkan-1-a64ec.dll"}
	default:
		return []string{"vulkan-1.dll", "vulkan-1-x64.dll", "vulkan-1-64.dll", "vulkan64.dll"}
	}
}

// versionedLoader reports a Vulkan runtime copy
// vulkan-1-<major>-<minor>-<patch>-<build>.dll. The installer writes
// these next to vulkan-1.dll, then copies the newest to vulkan-1.dll.
// vulkan-1-999-0-0-0.dll is the same loader under the GPU-PV placeholder name.
func versionedLoader(name string) bool {
	_, ok := versionRank(name)
	return ok
}

func versionRank(name string) ([4]int, bool) {
	base := strings.ToLower(winBase(name))
	const prefix = "vulkan-1-"
	if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, ".dll") {
		return [4]int{}, false
	}
	mid := strings.TrimSuffix(strings.TrimPrefix(base, prefix), ".dll")
	parts := strings.Split(mid, "-")
	if len(parts) != 4 {
		return [4]int{}, false
	}
	var rank [4]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return [4]int{}, false
		}
		rank[i] = n
	}
	return rank, true
}

// preferVersioned keeps versioned loader names, newest first.
// The 999.0.0.0 placeholder stays after a real version.
func preferVersioned(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if versionedLoader(name) {
			out = append(out, name)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return versionNewer(out[i], out[j])
	})
	return out
}

func versionNewer(a, b string) bool {
	ra, oka := versionRank(a)
	rb, okb := versionRank(b)
	if !oka || !okb {
		return oka && !okb
	}
	aPlace := ra == [4]int{999, 0, 0, 0}
	bPlace := rb == [4]int{999, 0, 0, 0}
	if aPlace != bPlace {
		return !aPlace
	}
	for i := range ra {
		if ra[i] != rb[i] {
			return ra[i] > rb[i]
		}
	}
	return false
}

func runtimeDir(goarch string) string {
	switch goarch {
	case "386":
		return "X86"
	case "arm64":
		return "ARM64"
	default:
		return "X64"
	}
}

func addSDK(add func(string), sdk, arch string) {
	sdk = strings.TrimRight(strings.TrimSpace(sdk), `\/`)
	if sdk == "" {
		return
	}
	if strings.EqualFold(filepath.Base(sdk), "vulkan-1.dll") {
		add(sdk)
		return
	}
	// VulkanRT's install folder and the components zip both keep a loader here.
	add(filepath.Join(sdk, "vulkan-1.dll"))
	add(filepath.Join(sdk, arch, "vulkan-1.dll"))
	add(filepath.Join(sdk, strings.ToLower(arch), "vulkan-1.dll"))
	add(filepath.Join(sdk, "Bin", "vulkan-1.dll"))
	add(filepath.Join(sdk, "RunTimeInstaller", arch, "vulkan-1.dll"))
	add(filepath.Join(sdk, "RunTimeInstaller", strings.ToLower(arch), "vulkan-1.dll"))
}

// sdkRoot is the install directory for a Vulkan uninstall entry.
// Other products are ignored. A quoted uninstall command is accepted
// when InstallLocation is empty.
func sdkRoot(displayName, installLocation, uninstallString string) string {
	if !strings.Contains(strings.ToLower(displayName), "vulkan") {
		return ""
	}
	loc := strings.TrimRight(strings.TrimSpace(installLocation), `\/`)
	if loc != "" {
		return loc
	}
	return commandDir(uninstallString)
}

func commandDir(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if text[0] == '"' {
		end := strings.Index(text[1:], `"`)
		if end >= 0 {
			text = text[1 : 1+end]
		}
	} else if i := strings.IndexAny(text, " \t"); i >= 0 {
		text = text[:i]
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = strings.TrimRight(text, `\/`)
	i := strings.LastIndexAny(text, `\/`)
	if i <= 0 {
		return ""
	}
	return strings.TrimRight(text[:i], `\/`)
}
