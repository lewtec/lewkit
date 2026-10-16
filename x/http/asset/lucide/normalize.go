package lucide

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"unicode"
)

var (
	errLucide = errors.New("lucide")
)

// guiSVG copies the root stroke onto each shape and swaps currentColor for c.
// line and polyline become paths. The SVG parser does not inherit paint from
// the root element, and it does not read those two tags.
func guiSVG(src string, c color.NRGBA) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(src))
	var root xml.StartElement
	var found bool
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("%w: svg: %w", errLucide, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "svg" {
			return "", fmt.Errorf("%w: root %s", errLucide, start.Name.Local)
		}
		root = start
		found = true
		break
	}
	if !found {
		return "", fmt.Errorf("%w: empty icon", errLucide)
	}
	kids, err := readShapes(dec)
	if err != nil {
		return "", err
	}
	width := attr(root.Attr, "width")
	height := attr(root.Attr, "height")
	viewBox := attr(root.Attr, "viewBox")
	if width == "" || height == "" || viewBox == "" {
		return "", fmt.Errorf("%w: icon missing width, height, or viewBox", errLucide)
	}
	inherited := map[string]string{
		"fill":            attr(root.Attr, "fill"),
		"stroke":          attr(root.Attr, "stroke"),
		"stroke-width":    attr(root.Attr, "stroke-width"),
		"stroke-linecap":  attr(root.Attr, "stroke-linecap"),
		"stroke-linejoin": attr(root.Attr, "stroke-linejoin"),
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
	b.WriteString(xmlEscape(width))
	b.WriteString(`" height="`)
	b.WriteString(xmlEscape(height))
	b.WriteString(`" viewBox="`)
	b.WriteString(xmlEscape(viewBox))
	b.WriteString(`">`)
	for _, kid := range kids {
		if err := writeShape(&b, kid, inherited, c); err != nil {
			return "", err
		}
	}
	b.WriteString("</svg>")
	return b.String(), nil
}

type shape struct {
	name  string
	attrs []xml.Attr
}

func readShapes(dec *xml.Decoder) ([]shape, error) {
	var kids []shape
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errLucide, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			kids = append(kids, shape{name: t.Name.Local, attrs: append([]xml.Attr(nil), t.Attr...)})
			if err := skipElement(dec); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if t.Name.Local != "svg" {
				return nil, fmt.Errorf("%w: unexpected </%s>", errLucide, t.Name.Local)
			}
			return kids, nil
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return nil, fmt.Errorf("%w: unexpected text", errLucide)
			}
		}
	}
}

func skipElement(dec *xml.Decoder) error {
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return nil
}

func writeShape(b *strings.Builder, kid shape, inherited map[string]string, c color.NRGBA) error {
	paint := map[string]string{}
	for key, val := range inherited {
		if val != "" {
			paint[key] = val
		}
	}
	geom := map[string]string{}
	for _, a := range kid.attrs {
		switch a.Name.Local {
		case "fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin":
			paint[a.Name.Local] = a.Value
		default:
			geom[a.Name.Local] = a.Value
		}
	}
	name := kid.name
	switch name {
	case "line":
		name = "path"
		geom["d"] = fmt.Sprintf("M %s %s L %s %s", num(geom["x1"]), num(geom["y1"]), num(geom["x2"]), num(geom["y2"]))
		delete(geom, "x1")
		delete(geom, "y1")
		delete(geom, "x2")
		delete(geom, "y2")
	case "polyline":
		name = "path"
		d, err := pointsPath(geom["points"])
		if err != nil {
			return err
		}
		geom["d"] = d
		delete(geom, "points")
	case "path", "circle", "ellipse", "rect", "polygon":
	default:
		return fmt.Errorf("%w: unknown tag %s", errLucide, kid.name)
	}
	recolor(paint, c)
	b.WriteByte('<')
	b.WriteString(name)
	writeAttrs(b, geom, geometryOrder(name))
	writeAttrs(b, paint, []string{"fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin", "stroke-opacity", "fill-opacity"})
	b.WriteString("/>")
	return nil
}

func geometryOrder(name string) []string {
	switch name {
	case "path":
		return []string{"d"}
	case "circle":
		return []string{"cx", "cy", "r"}
	case "ellipse":
		return []string{"cx", "cy", "rx", "ry"}
	case "rect":
		return []string{"x", "y", "width", "height", "rx", "ry"}
	case "polygon":
		return []string{"points"}
	default:
		return nil
	}
}

func writeAttrs(b *strings.Builder, vals map[string]string, order []string) {
	for _, key := range order {
		val, ok := vals[key]
		if !ok || val == "" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(key)
		b.WriteString(`="`)
		b.WriteString(xmlEscape(val))
		b.WriteByte('"')
	}
}

func recolor(paint map[string]string, c color.NRGBA) {
	hex := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	for _, key := range []string{"fill", "stroke"} {
		if strings.EqualFold(paint[key], "currentColor") {
			paint[key] = hex
		}
	}
	if c.A == 255 {
		return
	}
	op := strconv.FormatFloat(float64(c.A)/255, 'f', -1, 64)
	if paint["stroke"] != "" && paint["stroke"] != "none" {
		paint["stroke-opacity"] = op
	}
	if paint["fill"] != "" && paint["fill"] != "none" {
		paint["fill-opacity"] = op
	}
}

func pointsPath(points string) (string, error) {
	fields := strings.FieldsFunc(points, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	if len(fields) < 4 || len(fields)%2 != 0 {
		return "", fmt.Errorf("%w: points %q", errLucide, points)
	}
	var b strings.Builder
	b.WriteString("M ")
	b.WriteString(fields[0])
	b.WriteByte(' ')
	b.WriteString(fields[1])
	for i := 2; i < len(fields); i += 2 {
		b.WriteString(" L ")
		b.WriteString(fields[i])
		b.WriteByte(' ')
		b.WriteString(fields[i+1])
	}
	return b.String(), nil
}

func num(v string) string {
	if v == "" {
		return "0"
	}
	return v
}

func xmlEscape(s string) string {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}

func attr(attrs []xml.Attr, key string) string {
	for _, a := range attrs {
		if a.Name.Local == key {
			return a.Value
		}
	}
	return ""
}
