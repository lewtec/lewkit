package apk

import (
	"archive/zip"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrAssembleAPK is a hand-built APK that could not be written.
var ErrAssembleAPK = errors.New("assemble apk")

const (
	classMain     = "lewkit.MainActivity"
	classSurface  = "lewkit.SurfaceActivity"
	classPage     = "lewkit.PageActivity"
	classProvider = "lewkit.FileProvider"

	compileSDK  = 35
	minSDK      = 26
	targetSDK   = 35
	manifestZip = "AndroidManifest.xml"
	dexZip      = "classes.dex"
	arscZip     = "resources.arsc"
)

// assembleGoAPK writes an APK around the jniLibs already in workDir.
// The host classes are the embedded dex. Gradle and the Android SDK are
// not consulted. The caller signs the file.
func assembleGoAPK(cfg Config, workDir, outAPK string) error {
	if len(hostDex) < 8 || string(hostDex[:4]) != "dex\n" {
		return fmt.Errorf("%w: %w", ErrAssembleAPK, ErrHostDex)
	}
	libs, err := readJNILibs(workDir)
	if err != nil {
		return err
	}
	icons, err := readMipmaps(filepath.Join(workDir, "app", "src", "main", "res"))
	if err != nil {
		return err
	}
	manifest, err := manifestXML(cfg)
	if err != nil {
		return err
	}
	manBin, err := encodeXML(manifest, true)
	if err != nil {
		return err
	}
	layoutBin, err := encodeXML(splashLayout(), true)
	if err != nil {
		return err
	}
	pathsBin, err := encodeXML(filePathsXML(), false)
	if err != nil {
		return err
	}
	netBin, err := encodeXML(networkXML(), false)
	if err != nil {
		return err
	}
	table, resFiles, err := resourceTable(cfg, icons, layoutBin, pathsBin, netBin)
	if err != nil {
		return err
	}
	arsc, err := table.encode()
	if err != nil {
		return err
	}
	files := []apkEntry{
		{name: manifestZip, data: manBin},
		{name: dexZip, data: hostDex},
		{name: arscZip, data: arsc},
	}
	names := make([]string, 0, len(resFiles))
	for name := range resFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		files = append(files, apkEntry{name: name, data: resFiles[name]})
	}
	sort.Slice(libs, func(i, j int) bool { return libs[i].name < libs[j].name })
	for _, lib := range libs {
		files = append(files, apkEntry{name: lib.name, data: lib.data})
	}
	return writeZip(outAPK, files)
}

type apkEntry struct {
	name string
	data []byte
}

type jniLib struct {
	name string
	data []byte
}

func readJNILibs(workDir string) ([]jniLib, error) {
	root := filepath.Join(workDir, "app", "src", "main", "jniLibs")
	abis, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("%w: jniLibs: %w", ErrAssembleAPK, err)
	}
	var out []jniLib
	for _, abi := range abis {
		if !abi.IsDir() {
			continue
		}
		dir := filepath.Join(root, abi.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".so") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			out = append(out, jniLib{
				name: "lib/" + abi.Name() + "/" + file.Name(),
				data: raw,
			})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no native library", ErrAssembleAPK)
	}
	return out, nil
}

func readMipmaps(resDir string) (map[uint16][]byte, error) {
	densityOf := map[string]uint16{
		"mipmap-mdpi":    160,
		"mipmap-hdpi":    240,
		"mipmap-xhdpi":   320,
		"mipmap-xxhdpi":  480,
		"mipmap-xxxhdpi": 640,
	}
	out := map[uint16][]byte{}
	for dir, density := range densityOf {
		path := filepath.Join(resDir, dir, "ic_launcher.png")
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out[density] = raw
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: launcher icon", ErrAssembleAPK)
	}
	return out, nil
}

func manifestXML(cfg Config) (xNode, error) {
	filters, err := parseFilters(cfg.Capabilities.AndroidIntentFilters())
	if err != nil {
		return xNode{}, err
	}
	main := xNode{
		name: "activity",
		attrs: []xAttr{
			aStr("name", classMain),
			aBool("exported", true),
			aHex("configChanges", "orientation|screenSize|keyboardHidden|uiMode|fontScale", cfgChanges),
			aInt("launchMode", "singleTask", launchSingleTask),
		},
		kids: []xNode{launcherFilter()},
	}
	main.kids = append(main.kids, filters...)
	app := xNode{
		name: "application",
		attrs: []xAttr{
			aBool("allowBackup", false),
			aBool("extractNativeLibs", true),
			aRef("icon", "@mipmap/ic_launcher", resIDIcon),
			aRef("label", "@string/app_name", resIDAppName),
			aRef("networkSecurityConfig", "@xml/network_security_config", resIDNetSec),
			aBool("supportsRtl", true),
			aRef("theme", "@style/Theme.Eletrocromo", resIDTheme),
			aBool("usesCleartextTraffic", true),
			aBool("debuggable", true),
		},
		kids: []xNode{
			main,
			{
				name: "activity",
				attrs: []xAttr{
					aStr("name", classSurface),
					aBool("exported", false),
					aHex("configChanges", "orientation|screenSize|keyboardHidden|uiMode|fontScale", cfgChanges),
				},
			},
			{
				name: "activity",
				attrs: []xAttr{
					aStr("name", classPage),
					aBool("exported", false),
					aInt("launchMode", "standard", launchStandard),
					aHex("configChanges", "orientation|screenSize|keyboardHidden|uiMode|fontScale", cfgChanges),
				},
			},
			{
				name: "provider",
				attrs: []xAttr{
					aStr("name", classProvider),
					aStr("authorities", cfg.PackageID+".fileprovider"),
					aBool("exported", false),
					aBool("grantUriPermissions", true),
				},
				kids: []xNode{{
					name: "meta-data",
					attrs: []xAttr{
						aStr("name", "android.support.FILE_PROVIDER_PATHS"),
						aRef("resource", "@xml/file_paths", resIDFilePaths),
					},
				}},
			},
		},
	}
	return xNode{
		name: "manifest",
		attrs: []xAttr{
			plainStr("package", cfg.PackageID),
			plainInt("platformBuildVersionCode", "35", compileSDK),
			plainStr("platformBuildVersionName", "15"),
			aInt("versionCode", fmt.Sprintf("%d", cfg.VersionCode), uint32(cfg.VersionCode)),
			aStr("versionName", cfg.VersionName),
			aInt("compileSdkVersion", "35", compileSDK),
			aStr("compileSdkVersionCodename", "15"),
		},
		kids: []xNode{
			{
				name: "uses-sdk",
				attrs: []xAttr{
					aInt("minSdkVersion", "26", minSDK),
					aInt("targetSdkVersion", "35", targetSDK),
				},
			},
			{name: "uses-permission", attrs: []xAttr{aStr("name", "android.permission.INTERNET")}},
			{name: "uses-permission", attrs: []xAttr{aStr("name", "android.permission.ACCESS_NETWORK_STATE")}},
			{name: "uses-permission", attrs: []xAttr{aStr("name", "android.permission.POST_NOTIFICATIONS")}},
			app,
		},
	}, nil
}

func launcherFilter() xNode {
	return xNode{
		name: "intent-filter",
		kids: []xNode{
			{name: "action", attrs: []xAttr{aStr("name", "android.intent.action.MAIN")}},
			{name: "category", attrs: []xAttr{aStr("name", "android.intent.category.LAUNCHER")}},
		},
	}
}

type filterXML struct {
	Filters []struct {
		Actions []struct {
			Name string `xml:"http://schemas.android.com/apk/res/android name,attr"`
		} `xml:"action"`
		Categories []struct {
			Name string `xml:"http://schemas.android.com/apk/res/android name,attr"`
		} `xml:"category"`
		Data []struct {
			Scheme string `xml:"http://schemas.android.com/apk/res/android scheme,attr"`
			Mime   string `xml:"http://schemas.android.com/apk/res/android mimeType,attr"`
		} `xml:"data"`
	} `xml:"intent-filter"`
}

func parseFilters(fragment string) ([]xNode, error) {
	if strings.TrimSpace(fragment) == "" {
		return nil, nil
	}
	var doc filterXML
	wrapped := `<wrap xmlns:android="http://schemas.android.com/apk/res/android">` + fragment + `</wrap>`
	if err := xml.Unmarshal([]byte(wrapped), &doc); err != nil {
		return nil, fmt.Errorf("%w: intent filter: %w", ErrAssembleAPK, err)
	}
	out := make([]xNode, 0, len(doc.Filters))
	for _, filter := range doc.Filters {
		node := xNode{name: "intent-filter"}
		for _, action := range filter.Actions {
			node.kids = append(node.kids, xNode{name: "action", attrs: []xAttr{aStr("name", action.Name)}})
		}
		for _, category := range filter.Categories {
			node.kids = append(node.kids, xNode{name: "category", attrs: []xAttr{aStr("name", category.Name)}})
		}
		for _, data := range filter.Data {
			var attrs []xAttr
			if data.Scheme != "" {
				attrs = append(attrs, aStr("scheme", data.Scheme))
			}
			if data.Mime != "" {
				attrs = append(attrs, aStr("mimeType", data.Mime))
			}
			node.kids = append(node.kids, xNode{name: "data", attrs: attrs})
		}
		out = append(out, node)
	}
	return out, nil
}

func splashLayout() xNode {
	return xNode{
		name: "RelativeLayout",
		attrs: []xAttr{
			aRef("id", "@+id/splash", resIDSplash),
			aInt("layout_width", "match_parent", dimenMatch),
			aInt("layout_height", "match_parent", dimenMatch),
			aAttrRef("background", "?android:attr/colorBackground", attrColorBackground),
			aDP("paddingStart", 32),
			aDP("paddingEnd", 32),
		},
		kids: []xNode{
			{
				name: "ImageView",
				attrs: []xAttr{
					aRef("id", "@+id/splash_logo", resIDSplashLogo),
					aDP("layout_width", 120),
					aDP("layout_height", 120),
					aBool("layout_centerInParent", true),
					aRef("contentDescription", "@string/app_name", resIDAppName),
					aInt("scaleType", "fitCenter", 3),
					aRef("src", "@mipmap/ic_launcher", resIDIcon),
				},
			},
			{
				name: "LinearLayout",
				attrs: []xAttr{
					aInt("layout_width", "match_parent", dimenMatch),
					aInt("layout_height", "wrap_content", dimenWrap),
					aRef("layout_below", "@id/splash_logo", resIDSplashLogo),
					aBool("layout_centerHorizontal", true),
					aDP("layout_marginTop", 24),
					aInt("gravity", "center_horizontal", 1),
					aInt("orientation", "vertical", 1),
				},
				kids: []xNode{
					{
						name: "ProgressBar",
						attrs: []xAttr{
							aRef("id", "@+id/splash_progress", resIDSplashProgress),
							{name: "style", raw: "?android:attr/progressBarStyle", kind: typeAttribute, data: attrProgressBarStyle},
							aInt("layout_width", "wrap_content", dimenWrap),
							aInt("layout_height", "wrap_content", dimenWrap),
						},
					},
					{
						name: "TextView",
						attrs: []xAttr{
							aRef("id", "@+id/splash_status", resIDSplashStatus),
							aInt("layout_width", "match_parent", dimenMatch),
							aInt("layout_height", "wrap_content", dimenWrap),
							aDP("layout_marginTop", 16),
							aInt("gravity", "center", 0x11),
							aRef("text", "@string/loading", resIDLoading),
							aAttrRef("textAppearance", "?android:attr/textAppearanceMedium", attrTextAppearanceMedium),
							aAttrRef("textColor", "?android:attr/textColorSecondary", attrTextColorSecondary),
						},
					},
					{
						name: "TextView",
						attrs: []xAttr{
							aRef("id", "@+id/splash_detail", resIDSplashDetail),
							aInt("layout_width", "match_parent", dimenMatch),
							aInt("layout_height", "wrap_content", dimenWrap),
							aDP("layout_marginTop", 8),
							aInt("gravity", "center", 0x11),
							aAttrRef("textAppearance", "?android:attr/textAppearanceSmall", attrTextAppearanceSmall),
							aAttrRef("textColor", "?android:attr/textColorTertiary", attrTextColorTertiary),
							aInt("visibility", "gone", 2),
						},
					},
					{
						name: "Button",
						attrs: []xAttr{
							aRef("id", "@+id/splash_retry", resIDSplashRetry),
							aInt("layout_width", "wrap_content", dimenWrap),
							aInt("layout_height", "wrap_content", dimenWrap),
							aDP("layout_marginTop", 20),
							aRef("text", "@string/retry", resIDRetry),
							aInt("visibility", "gone", 2),
						},
					},
				},
			},
		},
	}
}

func filePathsXML() xNode {
	return xNode{
		name: "paths",
		kids: []xNode{
			{name: "cache-path", attrs: []xAttr{plainStr("name", "cache"), plainStr("path", ".")}},
			{name: "files-path", attrs: []xAttr{plainStr("name", "files"), plainStr("path", ".")}},
		},
	}
}

func networkXML() xNode {
	return xNode{
		name: "network-security-config",
		kids: []xNode{
			{
				name:  "domain-config",
				attrs: []xAttr{plainStr("cleartextTrafficPermitted", "true")},
				kids: []xNode{
					{name: "domain", attrs: []xAttr{plainStr("includeSubdomains", "false")}, text: "127.0.0.1"},
					{name: "domain", attrs: []xAttr{plainStr("includeSubdomains", "false")}, text: "localhost"},
				},
			},
			{name: "base-config", attrs: []xAttr{plainStr("cleartextTrafficPermitted", "false")}},
		},
	}
}

func resourceTable(cfg Config, icons map[uint16][]byte, layout, paths, net []byte) (arscTable, map[string][]byte, error) {
	var pool []string
	add := func(s string) uint32 {
		idx, next := strValue(pool, s)
		pool = next
		return idx
	}
	appName := add(cfg.AppName)
	loading := add("Starting…")
	retry := add("Try again")
	failed := add("Could not start the app")
	webErr := add("Could not load the page")
	webHTTP := add("Server returned HTTP %1$d")
	layoutPath := add("res/layout/activity_main.xml")
	pathsPath := add("res/xml/file_paths.xml")
	netPath := add("res/xml/network_security_config.xml")

	resFiles := map[string][]byte{
		"res/layout/activity_main.xml":        layout,
		"res/xml/file_paths.xml":              paths,
		"res/xml/network_security_config.xml": net,
	}
	densities := make([]uint16, 0, len(icons))
	for density := range icons {
		densities = append(densities, density)
	}
	sort.Slice(densities, func(i, j int) bool { return densities[i] < densities[j] })
	iconPaths := make([]uint32, len(densities))
	for i, density := range densities {
		name := fmt.Sprintf("res/mipmap-%s/ic_launcher.png", densityName(density))
		iconPaths[i] = add(name)
		resFiles[name] = icons[density]
	}
	// Default density uses the first bucket so a device always has a match.
	defaultIcon := iconPaths[0]

	present := func(v arscValue) arscConfig {
		return arscConfig{values: []arscValue{v}, missing: []bool{false}}
	}

	ids := []arscEntry{
		{name: "splash"},
		{name: "splash_detail"},
		{name: "splash_logo"},
		{name: "splash_progress"},
		{name: "splash_retry"},
		{name: "splash_status"},
	}
	idValues := make([]arscValue, len(ids))
	idMissing := make([]bool, len(ids))
	for i := range idValues {
		idValues[i] = simpleBool()
	}

	iconConfigs := []arscConfig{{
		values:  []arscValue{simpleStr(defaultIcon)},
		missing: []bool{false},
	}}
	for i, density := range densities {
		iconConfigs = append(iconConfigs, arscConfig{
			density: density,
			values:  []arscValue{simpleStr(iconPaths[i])},
			missing: []bool{false},
		})
	}

	strings := []struct {
		name string
		idx  uint32
	}{
		{"app_name", appName},
		{"loading", loading},
		{"retry", retry},
		{"server_failed", failed},
		{"webview_error", webErr},
		{"webview_http_error", webHTTP},
	}
	stringEntries := make([]arscEntry, len(strings))
	stringValues := make([]arscValue, len(strings))
	stringMissing := make([]bool, len(strings))
	for i, s := range strings {
		stringEntries[i] = arscEntry{name: s.name}
		stringValues[i] = simpleStr(s.idx)
	}

	table := arscTable{
		name:    cfg.PackageID,
		strings: pool,
		types: []arscType{
			{
				id:      1,
				name:    "id",
				entries: ids,
				configs: []arscConfig{{values: idValues, missing: idMissing}},
			},
			{
				id:      2,
				name:    "layout",
				entries: []arscEntry{{name: "activity_main"}},
				configs: []arscConfig{present(simpleStr(layoutPath))},
			},
			{
				id:      3,
				name:    "mipmap",
				entries: []arscEntry{{name: "ic_launcher"}},
				configs: iconConfigs,
			},
			{
				id:      4,
				name:    "string",
				entries: stringEntries,
				configs: []arscConfig{{values: stringValues, missing: stringMissing}},
			},
			{
				id:      5,
				name:    "style",
				entries: []arscEntry{{name: "Theme.Eletrocromo"}},
				configs: []arscConfig{
					{
						values: []arscValue{{
							bag:    true,
							parent: styleDeviceDefaultNoActionBar,
						}},
						missing: []bool{false},
					},
					{
						sdk: 29,
						values: []arscValue{{
							bag:    true,
							parent: styleDeviceDefaultDayNight,
							items: []arscBag{
								{name: attrWindowNoTitle, kind: typeIntBoolean, data: boolTrue},
								{name: attrWindowActionBar, kind: typeIntBoolean, data: boolFalse},
							},
						}},
						missing: []bool{false},
					},
				},
			},
			{
				id:   6,
				name: "xml",
				entries: []arscEntry{
					{name: "file_paths"},
					{name: "network_security_config"},
				},
				configs: []arscConfig{{
					values:  []arscValue{simpleStr(pathsPath), simpleStr(netPath)},
					missing: []bool{false, false},
				}},
			},
		},
	}
	return table, resFiles, nil
}

func densityName(density uint16) string {
	switch density {
	case 160:
		return "mdpi"
	case 240:
		return "hdpi"
	case 320:
		return "xhdpi"
	case 480:
		return "xxhdpi"
	case 640:
		return "xxxhdpi"
	default:
		return fmt.Sprintf("%d", density)
	}
}

func writeZip(path string, files []apkEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	cw := &countingWriter{w: f}
	zw := zip.NewWriter(cw)
	// Every entry is stored. Deflate would flush trailing bytes inside
	// the next CreateHeader, after the alignment pad was chosen.
	// archive/zip also writes a 16-byte data descriptor when the
	// previous entry closes, which is that same call. Modified stays
	// unset so no timestamp extra is appended after the pad.
	// MS-DOS time 0 is 1980-01-01.
	const zipDataDesc = 16
	var pending int
	var writeErr error
	for _, file := range files {
		hdr := &zip.FileHeader{
			Name:   file.name,
			Method: zip.Store,
			Extra:  alignExtra(int(cw.n)+pending, len(file.name), 4),
		}
		pending = zipDataDesc
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			writeErr = err
			break
		}
		if _, err := w.Write(file.data); err != nil {
			writeErr = err
			break
		}
	}
	closeErr := zw.Close()
	fileErr := f.Close()
	if writeErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("%w: %w", ErrAssembleAPK, writeErr)
	}
	if closeErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("%w: %w", ErrAssembleAPK, closeErr)
	}
	if fileErr != nil {
		os.Remove(tmp)
		return fileErr
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func alignExtra(offset, nameLen, align int) []byte {
	base := offset + 30 + nameLen
	pad := (align - (base % align)) % align
	if pad == 0 {
		return nil
	}
	if pad < 4 {
		pad += align
	}
	buf := make([]byte, pad)
	binary.LittleEndian.PutUint16(buf[2:], uint16(pad-4))
	return buf
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
