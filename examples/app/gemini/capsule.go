package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/lewtec/lewkit/x/gemini"
)

const defaultHomeURL = "gemini://home/"

type capsuleEntry struct {
	url     string
	title   string
	snippet string
	body    string
}

var builtinPages = []capsuleEntry{
	{
		url:     "gemini://home/",
		title:   "Geminispace Observatory",
		snippet: "Start screen, curated capsules, and interactive protocol demos.",
		body: `# Geminispace Observatory
## Welcome to the lewkit Gemini Browser

A lightweight, privacy-first hypertext browser rendered on the GPU with x/ui/gui and driven by Flutter-style tween & spring animations.

### Built-in Capsules
=> gemini://home/spec Protocol & Gemtext Specification
=> gemini://home/animations Flutter-Style Animation Showcase
=> gemini://home/search Search the Observatory Index
=> gemini://home/poetry Starlight & Small-Web Haiku
=> gemini://home/redirect Redirect Hop Demo (Status 31)

### Live Geminispace (TLS Port 1965)
=> gemini://geminiprotocol.net/ Official Gemini Protocol Capsule
=> gemini://kennedy.gemi.dev/ Kennedy Search Engine

### Why Gemini?
* No tracking scripts, cookies, or layout bloat
* Every response fits a single round-trip header plus UTF-8 gemtext
* Clean typography, hierarchy, and keyboard-first navigation
* Smooth 60+ FPS transitions powered by x/ui/gui curves and springs

> Per aspera ad astra — small protocols leave room for craft.
`,
	},
	{
		url:     "gemini://home/spec",
		title:   "Protocol & Gemtext Specification",
		snippet: "Overview of Gemini status codes, request framing, and text/gemini line types.",
		body: `# Gemini Protocol & Gemtext
## Request and Response Framing

A Gemini client connects over TLS (default port 1965) and sends a single CRLF-terminated URL:

` + "```gemini-request\ngemini://example.org/index.gmi\\r\\n\n```" + `

The server replies with a two-digit status code, a space, a meta string up to 1024 bytes, and CRLF:

* 10..19 — INPUT: Meta is a prompt; client re-requests with ?query
* 20..29 — SUCCESS: Meta is the MIME type (default text/gemini); body follows
* 30..39 — REDIRECT: Meta is the new target URL
* 40..49 — TEMPORARY FAILURE: Meta describes the error
* 50..59 — PERMANENT FAILURE: Not found or bad request
* 60..69 — CLIENT CERTIFICATE REQUIRED

## Gemtext Line Types

### Headings (#, ##, ###)
Three heading levels structure a document and form its outline.

### Links (=>)
=> gemini://home/ Return to Geminispace Observatory
=> gemini://home/animations Explore the Animation Architecture

### Lists, Quotes, and Preformatted Blocks
* Bullet items start with an asterisk and a space
* Quotes start with a greater-than sign
> Simplicity is prerequisite for reliability. — Edsger W. Dijkstra

` + "```go\ndoc := gemini.ParseGemtext(body)\nfor _, line := range doc.Lines {\n\tprintln(line.Kind.String(), line.Text)\n}\n```" + `
`,
	},
	{
		url:     "gemini://home/animations",
		title:   "Flutter-Style Animation Showcase",
		snippet: "How Tween, Curve, Spring, Opacity, and Slide compose in x/ui/gui.",
		body: `# Flutter-Style Animations in x/ui/gui
## Composable Motion Primitives

Every motion in this browser is declarative: Update steps tweens and springs on each TickMsg, and View wraps layout subtrees in Opacity, Slide, and ColumnView nodes.

### Foundation Primitives in x/ui/gui
* gui.Curve & gui.Cubic — EaseOutCubic, FastOutSlowIn, EaseOutBack, EaseInOut
* gui.Tween — Duration + Curve scalar interpolation with seamless retargeting
* gui.Spring — Critically damped harmonic oscillator for natural scroll physics
* gui.Opacity — Multiplies fill and glyph alpha across an entire Node subtree
* gui.Slide — Translates a child by pixel or fractional offsets without relayout
* gui.ColumnView — Clipped vertical viewport with proportional scrollbar thumb

` + "```go\npage := &gui.Opacity{\n\tValue: m.routeFade.Value(),\n\tChild: &gui.Slide{\n\t\tFractional: gui.Offset{X: m.routeSlide.Value()},\n\t\tChild:      m.documentView(),\n\t},\n}\n```" + `

### Try It Live
* Navigate between links to see directional Slide + Opacity page transitions
* Open the Bookmarks & History drawer (top-right button) to see FastOutSlowIn slide-over
* Scroll with the mouse wheel or Up/Down/PageDown keys to feel the Spring physics
* Toggle Light/Dark mode to watch smooth LerpRGB palette cross-fades

=> gemini://home/ Back to Observatory Home
=> gemini://home/search Try the Animated Status-10 Input Sheet
`,
	},
	{
		url:     "gemini://home/poetry",
		title:   "Starlight & Small-Web Haiku",
		snippet: "Short poems celebrating quiet networks and glowing phosphor.",
		body: `# Starlight & Small-Web Haiku
## Quiet Packets

> One socket opens,
> TLS handshake in the dark,
> Plain text blooms in light.

> No scripts on the wire,
> Just headings, links, and stillness,
> Thoughts arrive whole.

` + "```ascii\n      .       *        .       .\n   *      .        *       .       *\n       .      ✦ GEMINI ✦      .\n   .       *        .       *      .\n```" + `

=> gemini://home/ Return to Observatory Home
=> gemini://home/spec Read the Specification
`,
	},
}

func newDefaultClient() *gemini.Client {
	return &gemini.Client{
		Hosts: map[string]gemini.Handler{
			"home": serveBuiltinCapsule,
		},
	}
}

func serveBuiltinCapsule(_ context.Context, target *url.URL) (gemini.Response, error) {
	path := strings.TrimSuffix(target.Path, "/")
	if path == "" {
		path = "/"
	}
	switch path {
	case "/":
		return gemini.Response{
			Status: 20,
			Meta:   "text/gemini; charset=utf-8",
			Body:   []byte(builtinPages[0].body),
		}, nil
	case "/spec":
		return gemini.Response{
			Status: 20,
			Meta:   "text/gemini; charset=utf-8",
			Body:   []byte(builtinPages[1].body),
		}, nil
	case "/animations":
		return gemini.Response{
			Status: 20,
			Meta:   "text/gemini; charset=utf-8",
			Body:   []byte(builtinPages[2].body),
		}, nil
	case "/poetry":
		return gemini.Response{
			Status: 20,
			Meta:   "text/gemini; charset=utf-8",
			Body:   []byte(builtinPages[3].body),
		}, nil
	case "/redirect":
		return gemini.Response{
			Status: 31,
			Meta:   "gemini://home/animations",
		}, nil
	case "/search":
		q := strings.TrimSpace(target.RawQuery)
		if q == "" {
			return gemini.Response{
				Status: 10,
				Meta:   "Enter a search keyword (e.g. spring, gemini, haiku, status):",
			}, nil
		}
		decoded, err := url.QueryUnescape(q)
		if err != nil {
			decoded = q
		}
		return gemini.Response{
			Status: 20,
			Meta:   "text/gemini; charset=utf-8",
			Body:   []byte(renderSearchResults(decoded)),
		}, nil
	default:
		return gemini.Response{
			Status: 51,
			Meta:   fmt.Sprintf("capsule page %q not found on gemini://home", target.Path),
		}, nil
	}
}

func renderSearchResults(query string) string {
	needle := strings.ToLower(strings.TrimSpace(query))
	var b strings.Builder
	fmt.Fprintf(&b, "# Search Results for %q\n", query)
	fmt.Fprintf(&b, "=> gemini://home/search New Search Query\n")
	fmt.Fprintf(&b, "=> gemini://home/ Back to Observatory Home\n\n")

	matches := 0
	for _, page := range builtinPages {
		if strings.Contains(strings.ToLower(page.title), needle) ||
			strings.Contains(strings.ToLower(page.snippet), needle) ||
			strings.Contains(strings.ToLower(page.body), needle) {
			matches++
			fmt.Fprintf(&b, "## %s\n", page.title)
			fmt.Fprintf(&b, "%s\n", page.snippet)
			fmt.Fprintf(&b, "=> %s Open %s\n\n", page.url, page.title)
		}
	}
	if matches == 0 {
		fmt.Fprintf(&b, "> No built-in pages matched %q. Try \"spring\", \"gemini\", or \"haiku\".\n", query)
	}
	return b.String()
}
