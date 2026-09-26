package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Capability parse/validate sentinels.
var (
	ErrCapabilityUnknown = errors.New("unknown capability")
	ErrCapabilityEmpty   = errors.New("capability is empty")
	ErrSchemeInvalid     = errors.New("invalid url scheme")
	ErrFileTypeInvalid   = errors.New("invalid file type")
)

var schemePat = regexp.MustCompile(`^[a-z][a-z0-9+\-.]*$`)

// reservedSchemes cannot be claimed as app handlers.
var reservedSchemes = map[string]struct{}{
	"http": {}, "https": {}, "file": {}, "ftp": {}, "data": {},
	"javascript": {}, "about": {}, "blob": {}, "ws": {}, "wss": {},
}

// Capabilities is the closed catalog from eletrocromo.json.
type Capabilities struct {
	URL   *URLCap   `json:"url,omitempty"`
	Files *FilesCap `json:"files,omitempty"`
	Share *ShareCap `json:"share,omitempty"`
}

// ShareCap enables Go-side share-out (system share sheet).
type ShareCap struct{}

// URLCap registers custom URL schemes.
type URLCap struct {
	Schemes []string `json:"schemes"`
}

// FilesCap registers open-with / share types.
type FilesCap struct {
	Types []FileType `json:"types"`
}

// FileType is one extension + mime pair.
type FileType struct {
	Ext  string `json:"ext"`
	MIME string `json:"mime"`
}

// Empty reports whether any capability is set.
func (c Capabilities) Empty() bool {
	return (c.URL == nil || len(c.URL.Schemes) == 0) &&
		(c.Files == nil || len(c.Files.Types) == 0) &&
		c.Share == nil
}

// ParseCapabilities decodes the capabilities object. Unknown keys fail.
func ParseCapabilities(raw json.RawMessage) (Capabilities, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return Capabilities{}, nil
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Capabilities{}, fmt.Errorf("capabilities: %w", err)
	}
	for k := range probe {
		switch k {
		case "url", "files", "share":
		default:
			return Capabilities{}, fmt.Errorf("%w: %q", ErrCapabilityUnknown, k)
		}
	}
	var c Capabilities
	if err := json.Unmarshal(raw, &c); err != nil {
		return Capabilities{}, fmt.Errorf("capabilities: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Capabilities{}, err
	}
	return c, nil
}

// Validate checks declared capabilities. Empty is valid.
func (c Capabilities) Validate() error {
	if c.URL != nil {
		if len(c.URL.Schemes) == 0 {
			return fmt.Errorf("url: %w", ErrCapabilityEmpty)
		}
		seen := map[string]struct{}{}
		for i, s := range c.URL.Schemes {
			s = strings.ToLower(strings.TrimSpace(s))
			s = strings.TrimSuffix(s, "://")
			if !schemePat.MatchString(s) {
				return fmt.Errorf("%w: %q", ErrSchemeInvalid, c.URL.Schemes[i])
			}
			if _, bad := reservedSchemes[s]; bad {
				return fmt.Errorf("%w: reserved scheme %q", ErrSchemeInvalid, s)
			}
			if _, ok := seen[s]; ok {
				return fmt.Errorf("%w: duplicate scheme %q", ErrSchemeInvalid, s)
			}
			seen[s] = struct{}{}
			c.URL.Schemes[i] = s
		}
	}
	if c.Files != nil {
		if len(c.Files.Types) == 0 {
			return fmt.Errorf("files: %w", ErrCapabilityEmpty)
		}
		for i, ft := range c.Files.Types {
			ext, mime, err := normalizeFileType(ft)
			if err != nil {
				return err
			}
			c.Files.Types[i] = FileType{Ext: ext, MIME: mime}
		}
	}
	return nil
}

func normalizeFileType(ft FileType) (ext, mime string, err error) {
	ext = strings.ToLower(strings.TrimSpace(ft.Ext))
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" || strings.ContainsAny(ext, `/\`) {
		return "", "", fmt.Errorf("%w: ext %q", ErrFileTypeInvalid, ft.Ext)
	}
	ext = "." + ext
	mime = strings.ToLower(strings.TrimSpace(ft.MIME))
	if mime == "" || !strings.Contains(mime, "/") || strings.ContainsAny(mime, " \t") {
		return "", "", fmt.Errorf("%w: mime %q", ErrFileTypeInvalid, ft.MIME)
	}
	return ext, mime, nil
}

// AndroidIntentFilters is extra <intent-filter> XML for MainActivity.
func (c Capabilities) AndroidIntentFilters() string {
	var b strings.Builder
	if c.URL != nil {
		for _, s := range c.URL.Schemes {
			b.WriteString(`
        <intent-filter>
            <action android:name="android.intent.action.VIEW" />
            <category android:name="android.intent.category.DEFAULT" />
            <category android:name="android.intent.category.BROWSABLE" />
            <data android:scheme="` + xmlAttr(s) + `" />
        </intent-filter>
`)
		}
	}
	if c.Files != nil {
		mimes := make([]string, 0, len(c.Files.Types)+2)
		seen := map[string]struct{}{}
		add := func(m string) {
			if _, ok := seen[m]; ok {
				return
			}
			seen[m] = struct{}{}
			mimes = append(mimes, m)
		}
		for _, ft := range c.Files.Types {
			add(ft.MIME)
			if typ, _, ok := strings.Cut(ft.MIME, "/"); ok && typ != "" && typ != "*" {
				add(typ + "/*")
			}
		}
		for _, mime := range mimes {
			b.WriteString(`
        <intent-filter>
            <action android:name="android.intent.action.VIEW" />
            <action android:name="android.intent.action.SEND" />
            <action android:name="android.intent.action.SEND_MULTIPLE" />
            <category android:name="android.intent.category.DEFAULT" />
            <data android:mimeType="` + xmlAttr(mime) + `" />
        </intent-filter>
`)
		}
	}
	return b.String()
}

// ShareActivationXML is the NSExtensionActivationRule dict body.
func (c Capabilities) ShareActivationXML() string {
	var b strings.Builder
	b.WriteString("\t\t\t\t<key>NSExtensionActivationSupportsFileWithMaxCount</key>\n")
	b.WriteString("\t\t\t\t<integer>20</integer>\n")
	b.WriteString("\t\t\t\t<key>NSExtensionActivationSupportsImageWithMaxCount</key>\n")
	b.WriteString("\t\t\t\t<integer>20</integer>\n")
	b.WriteString("\t\t\t\t<key>NSExtensionActivationSupportsMovieWithMaxCount</key>\n")
	b.WriteString("\t\t\t\t<integer>8</integer>\n")
	b.WriteString("\t\t\t\t<key>NSExtensionActivationSupportsText</key>\n")
	b.WriteString("\t\t\t\t<true/>\n")
	b.WriteString("\t\t\t\t<key>NSExtensionActivationSupportsWebURLWithMaxCount</key>\n")
	b.WriteString("\t\t\t\t<integer>8</integer>\n")
	return b.String()
}

// PlistURLTypes is CFBundleURLTypes XML, or empty.
func (c Capabilities) PlistURLTypes(packageID string) string {
	var schemes []string
	if c.URL != nil {
		schemes = append(schemes, c.URL.Schemes...)
	}
	if len(schemes) == 0 && (c.Files != nil || c.Share != nil) {
		schemes = []string{c.WakeScheme(packageID)}
	}
	if len(schemes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\t<key>CFBundleURLTypes</key>\n\t<array>\n\t\t<dict>\n")
	b.WriteString("\t\t\t<key>CFBundleURLName</key>\n\t\t\t<string>")
	b.WriteString(XMLEscape(packageID))
	b.WriteString("</string>\n\t\t\t<key>CFBundleURLSchemes</key>\n\t\t\t<array>\n")
	for _, s := range schemes {
		b.WriteString("\t\t\t\t<string>")
		b.WriteString(XMLEscape(s))
		b.WriteString("</string>\n")
	}
	b.WriteString("\t\t\t</array>\n\t\t</dict>\n\t</array>\n")
	return b.String()
}

// PlistDocumentTypes is CFBundleDocumentTypes XML, or empty.
func (c Capabilities) PlistDocumentTypes() string {
	if c.Files == nil || len(c.Files.Types) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\t<key>CFBundleDocumentTypes</key>\n\t<array>\n")
	imported := map[string]FileType{}
	for _, ft := range c.Files.Types {
		name := strings.TrimPrefix(ft.Ext, ".")
		uti := utiFor(ft.MIME)
		b.WriteString("\t\t<dict>\n")
		b.WriteString("\t\t\t<key>CFBundleTypeName</key>\n\t\t\t<string>")
		b.WriteString(XMLEscape(name))
		b.WriteString("</string>\n")
		b.WriteString("\t\t\t<key>CFBundleTypeRole</key>\n\t\t\t<string>Viewer</string>\n")
		b.WriteString("\t\t\t<key>LSHandlerRank</key>\n\t\t\t<string>Alternate</string>\n")
		b.WriteString("\t\t\t<key>CFBundleTypeExtensions</key>\n\t\t\t<array>\n")
		b.WriteString("\t\t\t\t<string>")
		b.WriteString(XMLEscape(name))
		b.WriteString("</string>\n\t\t\t</array>\n")
		b.WriteString("\t\t\t<key>LSItemContentTypes</key>\n\t\t\t<array>\n")
		b.WriteString("\t\t\t\t<string>")
		b.WriteString(XMLEscape(uti))
		b.WriteString("</string>\n")
		if strings.HasPrefix(ft.MIME, "text/") && uti != "public.text" {
			b.WriteString("\t\t\t\t<string>public.text</string>\n")
		}
		if strings.HasPrefix(ft.MIME, "image/") && uti != "public.image" {
			b.WriteString("\t\t\t\t<string>public.image</string>\n")
		}
		b.WriteString("\t\t\t</array>\n")
		b.WriteString("\t\t</dict>\n")
		if !systemUTI(uti) {
			imported[uti] = ft
		}
	}
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>LSSupportsOpeningDocumentsInPlace</key>\n\t<false/>\n")
	if len(imported) > 0 {
		b.WriteString("\t<key>UTImportedTypeDeclarations</key>\n\t<array>\n")
		utis := make([]string, 0, len(imported))
		for uti := range imported {
			utis = append(utis, uti)
		}
		slices.Sort(utis)
		for _, uti := range utis {
			ft := imported[uti]
			name := strings.TrimPrefix(ft.Ext, ".")
			b.WriteString("\t\t<dict>\n")
			b.WriteString("\t\t\t<key>UTTypeIdentifier</key>\n\t\t\t<string>")
			b.WriteString(XMLEscape(uti))
			b.WriteString("</string>\n")
			b.WriteString("\t\t\t<key>UTTypeDescription</key>\n\t\t\t<string>")
			b.WriteString(XMLEscape(name))
			b.WriteString("</string>\n")
			b.WriteString("\t\t\t<key>UTTypeConformsTo</key>\n\t\t\t<array>\n")
			b.WriteString("\t\t\t\t<string>public.plain-text</string>\n")
			b.WriteString("\t\t\t</array>\n")
			b.WriteString("\t\t\t<key>UTTypeTagSpecification</key>\n\t\t\t<dict>\n")
			b.WriteString("\t\t\t\t<key>public.filename-extension</key>\n\t\t\t\t<array>\n")
			b.WriteString("\t\t\t\t\t<string>")
			b.WriteString(XMLEscape(name))
			b.WriteString("</string>\n\t\t\t\t</array>\n")
			b.WriteString("\t\t\t\t<key>public.mime-type</key>\n\t\t\t\t<array>\n")
			b.WriteString("\t\t\t\t\t<string>")
			b.WriteString(XMLEscape(ft.MIME))
			b.WriteString("</string>\n\t\t\t\t</array>\n")
			b.WriteString("\t\t\t</dict>\n")
			b.WriteString("\t\t</dict>\n")
		}
		b.WriteString("\t</array>\n")
	}
	return b.String()
}

func systemUTI(uti string) bool {
	return strings.HasPrefix(uti, "public.") ||
		uti == "com.adobe.pdf"
}

// WakeScheme is the custom URL used to foreground the app after a share.
func (c Capabilities) WakeScheme(packageID string) string {
	if c.URL != nil && len(c.URL.Schemes) > 0 {
		return c.URL.Schemes[0]
	}
	parts := strings.Split(packageID, ".")
	return "eletrocromo-" + parts[len(parts)-1]
}

// AppGroupID is the shared container for the iOS share extension.
func AppGroupID(packageID string) string {
	return "group." + packageID
}

func xmlAttr(s string) string {
	return XMLEscape(s)
}

func utiFor(mime string) string {
	switch mime {
	case "application/pdf":
		return "com.adobe.pdf"
	case "text/plain":
		return "public.plain-text"
	case "text/markdown":
		return "net.daringfireball.markdown"
	case "image/png":
		return "public.png"
	case "image/jpeg", "image/jpg":
		return "public.jpeg"
	case "image/gif":
		return "public.gif"
	case "image/heic", "image/heif":
		return "public.heic"
	case "image/webp":
		return "org.webmproject.webp"
	case "image/*":
		return "public.image"
	default:
		return "public.data"
	}
}
