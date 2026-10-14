package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCapabilities_UnknownKey(t *testing.T) {
	_, err := ParseCapabilities([]byte(`{"camera":{"usage":"x"}}`))
	require.ErrorIs(t, err, ErrCapabilityUnknown)
}

func TestParseCapabilities_URLAndFiles(t *testing.T) {
	c, err := ParseCapabilities([]byte(`{
		"url":{"schemes":["MyApp"]},
		"files":{"types":[{"ext":"md","mime":"text/markdown"}]}
	}`))
	require.NoError(t, err)
	if len(c.URL.Schemes) != 1 || c.URL.Schemes[0] != "myapp" {
		require.FailNowf(t, "", "schemes = %#v", c.URL.Schemes)
	}
	if c.Files.Types[0].Ext != ".md" || c.Files.Types[0].MIME != "text/markdown" {
		require.FailNowf(t, "", "types = %#v", c.Files.Types)
	}
}

func TestValidate_ReservedScheme(t *testing.T) {
	c := Capabilities{URL: &URLCap{Schemes: []string{"https"}}}
	err := c.Validate()
	require.ErrorIs(t, err, ErrSchemeInvalid)
}

func TestAndroidIntentFilters(t *testing.T) {
	c := Capabilities{
		URL:   &URLCap{Schemes: []string{"myapp"}},
		Files: &FilesCap{Types: []FileType{{Ext: ".pdf", MIME: "application/pdf"}}},
	}
	s := c.AndroidIntentFilters()
	if !strings.Contains(s, `android:scheme="myapp"`) {
		require.FailNowf(t, "", "scheme:\n%s", s)
	}
	if !strings.Contains(s, `android:mimeType="application/pdf"`) {
		require.FailNowf(t, "", "mime:\n%s", s)
	}
	img := Capabilities{Files: &FilesCap{Types: []FileType{{Ext: ".jpg", MIME: "image/jpeg"}}}}
	is := img.AndroidIntentFilters()
	if !strings.Contains(is, `android:mimeType="image/*"`) || !strings.Contains(is, "SEND_MULTIPLE") {
		require.FailNowf(t, "", "image share:\n%s", is)
	}
}

func TestPlistFragments(t *testing.T) {
	c := Capabilities{
		URL:   &URLCap{Schemes: []string{"myapp"}},
		Files: &FilesCap{Types: []FileType{{Ext: ".md", MIME: "text/markdown"}}},
	}
	u := c.PlistURLTypes("br.tec.lew.demo")
	if !strings.Contains(u, "CFBundleURLTypes") || !strings.Contains(u, "myapp") {
		require.FailNowf(t, "", "url types:\n%s", u)
	}
	d := c.PlistDocumentTypes()
	if !strings.Contains(d, "CFBundleDocumentTypes") || !strings.Contains(d, "md") {
		require.FailNowf(t, "", "docs:\n%s", d)
	}
	if !strings.Contains(d, "LSSupportsOpeningDocumentsInPlace") {
		require.FailNowf(t, "", "in-place:\n%s", d)
	}
	if !strings.Contains(d, "UTImportedTypeDeclarations") || !strings.Contains(d, "LSHandlerRank") {
		require.FailNowf(t, "", "imported/rank:\n%s", d)
	}
}
