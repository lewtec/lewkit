package cmdpalette

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNormalizesAndDropsBad(t *testing.T) {
	p := New([]Item{
		{Label: "A", Href: "/a", Group: "G"},
		{Label: "skip"}, // no href
		ItemOf("Q", "All", "/q", "list"),
	})
	require.Len(t, p.Items, 2)
	require.NotEmpty(t, p.Items[0].Keywords)
	require.Contains(t, p.Items[1].Keywords, "list")
}

func TestKeywordsColon(t *testing.T) {
	k := Keywords("account", "Assets:Cash")
	require.Contains(t, k, "cash")
}

func TestButtonAndModalRender(t *testing.T) {
	p := New([]Item{ItemOf("Reports", "Check", "/l/x/check")}).
		WithID("jump").
		WithTitle("Jump").
		WithButtonClass("btn btn-xs")
	var buf bytes.Buffer
	require.NoError(t, p.Button().Render(t.Context(), &buf))
	html := buf.String()
	for _, want := range []string{`id="jump-open"`, `aria-controls="jump"`, "Jump", "btn btn-xs"} {
		require.Contains(t, html, want)
	}
	buf.Reset()
	require.NoError(t, p.Modal().Render(t.Context(), &buf))
	html = buf.String()
	for _, want := range []string{
		`id="jump"`,
		`data-cmdpalette`,
		`id="jump-list"`,
		`data-cmd-item`,
		"/l/x/check",
		"Check",
		"Reports",
		"cmdpalette-list", // single-column class, not daisy menu
	} {
		require.Contains(t, html, want)
	}
	require.NotContains(t, html, `class="menu`)
}

func TestDaisyModalUsesMenu(t *testing.T) {
	p := New([]Item{ItemOf("Artifact", "branch.elf", "/artifact/branch")}).
		WithTitle("Jump").
		WithButtonClass("btn btn-sm")
	var buf bytes.Buffer
	require.NoError(t, p.DaisyButton().Render(t.Context(), &buf))
	html := buf.String()
	for _, want := range []string{`id="cmdpalette-open"`, `class="btn btn-sm"`, "Jump", "Ctrl+K"} {
		require.Contains(t, html, want)
	}
	buf.Reset()
	require.NoError(t, p.DaisyModal().Render(t.Context(), &buf))
	html = buf.String()
	for _, want := range []string{
		`data-cmdpalette`,
		`class="modal"`,
		`class="menu menu-sm max-h-80 w-full flex-nowrap overflow-y-auto"`,
		`data-keywords=`,
		`href="/artifact/branch"`,
	} {
		require.Contains(t, html, want)
	}
	keywords := strings.Index(html, "data-keywords=")
	href := strings.Index(html, `href="`)
	require.GreaterOrEqual(t, keywords, 0)
	require.GreaterOrEqual(t, href, keywords)
}

func TestAssetsUnescaped(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Assets().Render(t.Context(), &buf))
	s := buf.String()
	require.Contains(t, s, "cmdpalette-dialog")
	require.Contains(t, s, "__cmdpaletteBound")
	require.NotContains(t, s, "cmdpalette-dialog&#")
}
