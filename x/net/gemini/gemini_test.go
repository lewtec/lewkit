package gemini

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndFormatGemtext(t *testing.T) {
	const raw = `# Capsule Welcome
## Subheading
### Minor heading

Welcome to Geminispace.
=> gemini://example.org/ Home Link
=> /docs/spec.gmi
* First item
* Second item
> A wise quote from the stars.
` + "```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n"

	doc := ParseGemtext(raw)
	assert.Equal(t, "Capsule Welcome", doc.Title())
	require.Len(t, doc.Lines, 11)
	assert.Equal(t, Line{Kind: KindHeading, Level: 1, Text: "Capsule Welcome"}, doc.Lines[0])
	assert.Equal(t, Line{Kind: KindHeading, Level: 2, Text: "Subheading"}, doc.Lines[1])
	assert.Equal(t, Line{Kind: KindHeading, Level: 3, Text: "Minor heading"}, doc.Lines[2])
	assert.Equal(t, Line{Kind: KindText, Text: ""}, doc.Lines[3])
	assert.Equal(t, Line{Kind: KindText, Text: "Welcome to Geminispace."}, doc.Lines[4])
	assert.Equal(t, Line{Kind: KindLink, URL: "gemini://example.org/", Text: "Home Link"}, doc.Lines[5])
	assert.Equal(t, Line{Kind: KindLink, URL: "/docs/spec.gmi", Text: "/docs/spec.gmi"}, doc.Lines[6])
	assert.Equal(t, Line{Kind: KindListItem, Text: "First item"}, doc.Lines[7])
	assert.Equal(t, Line{Kind: KindListItem, Text: "Second item"}, doc.Lines[8])
	assert.Equal(t, Line{Kind: KindQuote, Text: "A wise quote from the stars."}, doc.Lines[9])
	assert.Equal(t, Line{Kind: KindPreformatted, Alt: "go", Text: "func main() {\n\tprintln(\"hi\")\n}"}, doc.Lines[10])

	roundtrip := ParseGemtext(FormatGemtext(doc))
	assert.Equal(t, doc, roundtrip)
}

func TestNormalizeAndResolveURL(t *testing.T) {
	u, err := NormalizeURL("geminiprotocol.net")
	require.NoError(t, err)
	assert.Equal(t, "gemini://geminiprotocol.net/", u.String())

	resolved, err := ResolveURL("gemini://capsule.local/docs/index.gmi", "../about.gmi")
	require.NoError(t, err)
	assert.Equal(t, "gemini://capsule.local/about.gmi", resolved)

	_, err = NormalizeURL("http://example.com")
	require.ErrorIs(t, err, ErrInvalidURL)
}

func TestReadResponseAndRedirects(t *testing.T) {
	resp, err := ReadResponse(strings.NewReader("20 text/gemini\r\n# Hello\nWorld\n"))
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Status)
	assert.Equal(t, StatusSuccess, Category(resp.Status))
	assert.Equal(t, "text/gemini", resp.Meta)
	assert.Equal(t, "# Hello\nWorld\n", string(resp.Body))

	client := &Client{
		Hosts: map[string]Handler{
			"local": func(_ context.Context, target *url.URL) (Response, error) {
				if target.Path == "/old" {
					return Response{Status: CodeRedirectPermanent, Meta: "/new"}, nil
				}
				return Response{Status: CodeSuccess, Meta: "text/gemini", Body: []byte("# Arrived\n")}, nil
			},
		},
	}
	got, err := client.Fetch(t.Context(), "gemini://local/old")
	require.NoError(t, err)
	assert.Equal(t, "gemini://local/new", got.URL)
	assert.Equal(t, CodeSuccess, got.Status)
	assert.Equal(t, "# Arrived\n", string(got.Body))
}

func TestFetchContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := (&Client{}).Fetch(ctx, "gemini://example.org/")
	require.ErrorIs(t, err, context.Canceled)
}
