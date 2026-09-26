package common

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCapabilities_UnknownKey(t *testing.T) {
	_, err := ParseCapabilities([]byte(`{"camera":{"usage":"x"}}`))
	if !errors.Is(err, ErrCapabilityUnknown) {
		t.Fatalf("err = %v; want ErrCapabilityUnknown", err)
	}
}

func TestParseCapabilities_URLAndFiles(t *testing.T) {
	c, err := ParseCapabilities([]byte(`{
		"url":{"schemes":["MyApp"]},
		"files":{"types":[{"ext":"md","mime":"text/markdown"}]}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.URL.Schemes) != 1 || c.URL.Schemes[0] != "myapp" {
		t.Fatalf("schemes = %#v", c.URL.Schemes)
	}
	if c.Files.Types[0].Ext != ".md" || c.Files.Types[0].MIME != "text/markdown" {
		t.Fatalf("types = %#v", c.Files.Types)
	}
}

func TestValidate_ReservedScheme(t *testing.T) {
	c := Capabilities{URL: &URLCap{Schemes: []string{"https"}}}
	if err := c.Validate(); !errors.Is(err, ErrSchemeInvalid) {
		t.Fatalf("err = %v", err)
	}
}

func TestAndroidIntentFilters(t *testing.T) {
	c := Capabilities{
		URL:   &URLCap{Schemes: []string{"myapp"}},
		Files: &FilesCap{Types: []FileType{{Ext: ".pdf", MIME: "application/pdf"}}},
	}
	s := c.AndroidIntentFilters()
	if !strings.Contains(s, `android:scheme="myapp"`) {
		t.Fatalf("scheme:\n%s", s)
	}
	if !strings.Contains(s, `android:mimeType="application/pdf"`) {
		t.Fatalf("mime:\n%s", s)
	}
	img := Capabilities{Files: &FilesCap{Types: []FileType{{Ext: ".jpg", MIME: "image/jpeg"}}}}
	is := img.AndroidIntentFilters()
	if !strings.Contains(is, `android:mimeType="image/*"`) || !strings.Contains(is, "SEND_MULTIPLE") {
		t.Fatalf("image share:\n%s", is)
	}
}

func TestPlistFragments(t *testing.T) {
	c := Capabilities{
		URL:   &URLCap{Schemes: []string{"myapp"}},
		Files: &FilesCap{Types: []FileType{{Ext: ".md", MIME: "text/markdown"}}},
	}
	u := c.PlistURLTypes("br.tec.lew.demo")
	if !strings.Contains(u, "CFBundleURLTypes") || !strings.Contains(u, "myapp") {
		t.Fatalf("url types:\n%s", u)
	}
	d := c.PlistDocumentTypes()
	if !strings.Contains(d, "CFBundleDocumentTypes") || !strings.Contains(d, "md") {
		t.Fatalf("docs:\n%s", d)
	}
	if !strings.Contains(d, "LSSupportsOpeningDocumentsInPlace") {
		t.Fatalf("in-place:\n%s", d)
	}
	if !strings.Contains(d, "UTImportedTypeDeclarations") || !strings.Contains(d, "LSHandlerRank") {
		t.Fatalf("imported/rank:\n%s", d)
	}
}
