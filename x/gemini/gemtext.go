package gemini

import (
	"fmt"
	"strings"
)

// Kind is the gemtext line kind.
//
//lewkit:enum
type Kind int

const (
	KindText Kind = iota + 1
	KindLink
	KindHeading
	KindListItem
	KindQuote
	KindPreformatted
)

// Line is one parsed gemtext block or line.
// For KindHeading, Level is 1, 2, or 3.
// For KindLink, URL is the target and Text is the optional label (or URL when omitted).
// For KindPreformatted, Alt is the opening fence caption and Text is the body lines joined by '\n'.
type Line struct {
	Kind  Kind
	Level int
	Text  string
	URL   string
	Alt   string
}

// Document is a parsed gemtext document.
type Document struct {
	Lines []Line
}

// Title returns the first heading in doc, or an empty string if none exists.
func (doc Document) Title() string {
	for _, line := range doc.Lines {
		if line.Kind == KindHeading && strings.TrimSpace(line.Text) != "" {
			return strings.TrimSpace(line.Text)
		}
	}
	return ""
}

// ParseGemtext parses a text/gemini body into a [Document].
func ParseGemtext(source string) Document {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	rawLines := strings.Split(source, "\n")
	var out []Line
	inPre := false
	var preAlt string
	var preLines []string

	for i, raw := range rawLines {
		if i == len(rawLines)-1 && raw == "" && !inPre && len(out) > 0 {
			break
		}
		if strings.HasPrefix(raw, "```") {
			if inPre {
				out = append(out, Line{
					Kind: KindPreformatted,
					Alt:  preAlt,
					Text: strings.Join(preLines, "\n"),
				})
				inPre = false
				preAlt = ""
				preLines = nil
			} else {
				inPre = true
				preAlt = strings.TrimSpace(strings.TrimPrefix(raw, "```"))
				preLines = nil
			}
			continue
		}
		if inPre {
			preLines = append(preLines, raw)
			continue
		}
		switch {
		case strings.HasPrefix(raw, "=>"):
			rest := strings.TrimSpace(strings.TrimPrefix(raw, "=>"))
			if rest == "" {
				out = append(out, Line{Kind: KindText})
				continue
			}
			target, label := splitLink(rest)
			if label == "" {
				label = target
			}
			out = append(out, Line{Kind: KindLink, URL: target, Text: label})
		case strings.HasPrefix(raw, "###"):
			out = append(out, Line{Kind: KindHeading, Level: 3, Text: strings.TrimSpace(strings.TrimPrefix(raw, "###"))})
		case strings.HasPrefix(raw, "##"):
			out = append(out, Line{Kind: KindHeading, Level: 2, Text: strings.TrimSpace(strings.TrimPrefix(raw, "##"))})
		case strings.HasPrefix(raw, "#"):
			out = append(out, Line{Kind: KindHeading, Level: 1, Text: strings.TrimSpace(strings.TrimPrefix(raw, "#"))})
		case strings.HasPrefix(raw, "* "):
			out = append(out, Line{Kind: KindListItem, Text: strings.TrimSpace(strings.TrimPrefix(raw, "* "))})
		case raw == "*":
			out = append(out, Line{Kind: KindListItem})
		case strings.HasPrefix(raw, ">"):
			out = append(out, Line{Kind: KindQuote, Text: strings.TrimSpace(strings.TrimPrefix(raw, ">"))})
		default:
			out = append(out, Line{Kind: KindText, Text: raw})
		}
	}
	if inPre {
		out = append(out, Line{
			Kind: KindPreformatted,
			Alt:  preAlt,
			Text: strings.Join(preLines, "\n"),
		})
	}
	return Document{Lines: out}
}

// FormatGemtext renders doc back into canonical text/gemini text.
func FormatGemtext(doc Document) string {
	var b strings.Builder
	for _, line := range doc.Lines {
		switch line.Kind {
		case KindHeading:
			hashes := "#"
			switch line.Level {
			case 2:
				hashes = "##"
			case 3:
				hashes = "###"
			}
			fmt.Fprintf(&b, "%s %s\n", hashes, line.Text)
		case KindLink:
			if line.Text != "" && line.Text != line.URL {
				fmt.Fprintf(&b, "=> %s %s\n", line.URL, line.Text)
			} else {
				fmt.Fprintf(&b, "=> %s\n", line.URL)
			}
		case KindListItem:
			fmt.Fprintf(&b, "* %s\n", line.Text)
		case KindQuote:
			fmt.Fprintf(&b, "> %s\n", line.Text)
		case KindPreformatted:
			fmt.Fprintf(&b, "```%s\n%s\n```\n", line.Alt, line.Text)
		default:
			fmt.Fprintf(&b, "%s\n", line.Text)
		}
	}
	return b.String()
}

func splitLink(rest string) (target, label string) {
	for i, r := range rest {
		if r == ' ' || r == '\t' {
			return rest[:i], strings.TrimSpace(rest[i:])
		}
	}
	return rest, ""
}
