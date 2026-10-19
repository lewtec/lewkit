package term

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"strconv"
	"strings"
)

// apcLimit is how much of one kitty graphics command the grid will keep.
const apcLimit = 8 << 20

// pictures are kitty images the process placed, drawn into cells.
type pictures struct {
	images     map[int]image.Image
	parts      map[int][]byte
	held       map[int]kittyCtl
	placements []placement
}

type placement struct {
	id         int
	col, row   int
	cols, rows int
}

type kittyCtl struct {
	action       rune
	format       int
	id           int
	more         int
	cols, rows   int
	pixW, pixH   int
	compression  rune
	transmission rune
	del          rune
}

func (g *Grid) apcByte(b byte) {
	if b == 0x1b {
		g.state = stateAPCEsc
		return
	}
	if b == 0x07 {
		g.finishAPC()
		return
	}
	if g.apcDrop {
		return
	}
	if len(g.apc) >= apcLimit {
		g.apcDrop = true
		g.apc = g.apc[:0]
		return
	}
	g.apc = append(g.apc, b)
}

func (g *Grid) apcEsc(b byte) {
	if b == '\\' {
		g.finishAPC()
		return
	}
	g.apc = g.apc[:0]
	g.apcDrop = false
	g.state = stateEsc
	g.esc(b)
}

func (g *Grid) finishAPC() {
	body := g.apc
	g.apc = g.apc[:0]
	dropped := g.apcDrop
	g.apcDrop = false
	g.state = stateText
	if dropped || len(body) == 0 || body[0] != 'G' {
		return
	}
	g.kitty(body[1:])
}

func (g *Grid) kitty(body []byte) {
	semi := bytes.IndexByte(body, ';')
	head := body
	var data []byte
	if semi >= 0 {
		head = body[:semi]
		data = body[semi+1:]
	}
	ctl := parseKitty(string(head))
	switch ctl.action {
	case 'd':
		g.deleteKitty(ctl)
	case 'p':
		g.placeKitty(ctl.id, ctl)
	case 't', 'T':
		if ctl.transmission != 0 && ctl.transmission != 'd' {
			return
		}
		if ctl.more == 1 {
			if g.pics.held == nil {
				g.pics.held = map[int]kittyCtl{}
			}
			if _, ok := g.pics.held[ctl.id]; !ok {
				g.pics.held[ctl.id] = ctl
			}
			g.kittyAppend(ctl, data)
			return
		}
		if held, ok := g.pics.held[ctl.id]; ok {
			ctl = mergeKitty(held, ctl)
			delete(g.pics.held, ctl.id)
		}
		full := append(g.kittyTake(ctl.id), data...)
		img := decodeKitty(ctl, full)
		if img == nil {
			return
		}
		if g.pics.images == nil {
			g.pics.images = map[int]image.Image{}
		}
		g.pics.images[ctl.id] = img
		if ctl.action == 'T' {
			g.placeKitty(ctl.id, ctl)
		}
	}
}

func (g *Grid) kittyAppend(ctl kittyCtl, data []byte) {
	if g.pics.parts == nil {
		g.pics.parts = map[int][]byte{}
	}
	next := append(g.pics.parts[ctl.id], data...)
	if len(next) > apcLimit {
		delete(g.pics.parts, ctl.id)
		return
	}
	g.pics.parts[ctl.id] = next
}

func (g *Grid) kittyTake(id int) []byte {
	part := g.pics.parts[id]
	delete(g.pics.parts, id)
	return part
}

func (g *Grid) placeKitty(id int, ctl kittyCtl) {
	img := g.pics.images[id]
	if img == nil {
		return
	}
	cols, rows := ctl.cols, ctl.rows
	bounds := img.Bounds()
	if cols <= 0 {
		cols = (bounds.Dx() + 7) / 8
	}
	if rows <= 0 {
		rows = (bounds.Dy() + 15) / 16
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	atC, atR := g.cursorC, g.cursorR
	if atC < 0 {
		atC = 0
	}
	if atR < 0 {
		atR = 0
	}
	if atC >= g.cols || atR >= g.rows {
		return
	}
	if atC+cols > g.cols {
		cols = g.cols - atC
	}
	if atR+rows > g.rows {
		rows = g.rows - atR
	}
	g.raster(img, atC, atR, cols, rows)
	g.pics.placements = append(g.pics.placements, placement{id: id, col: atC, row: atR, cols: cols, rows: rows})
}

func (g *Grid) deleteKitty(ctl kittyCtl) {
	if ctl.del == 'a' || (ctl.del == 0 && ctl.id == 0) {
		for _, item := range g.pics.placements {
			g.blankRect(item.col, item.row, item.cols, item.rows)
		}
		g.pics = pictures{}
		return
	}
	id := ctl.id
	kept := g.pics.placements[:0]
	for _, item := range g.pics.placements {
		if item.id == id {
			g.blankRect(item.col, item.row, item.cols, item.rows)
			continue
		}
		kept = append(kept, item)
	}
	g.pics.placements = kept
	delete(g.pics.images, id)
	delete(g.pics.parts, id)
}

func (g *Grid) blankRect(col, row, cols, rows int) {
	empty := g.erased()
	for y := row; y < row+rows; y++ {
		if y < 0 || y >= len(g.screen) {
			continue
		}
		for x := col; x < col+cols; x++ {
			if x < 0 || x >= len(g.screen[y].cells) {
				continue
			}
			g.screen[y].cells[x] = empty
		}
	}
}

func (g *Grid) raster(img image.Image, atC, atR, cols, rows int) {
	bounds := img.Bounds()
	pw, ph := bounds.Dx(), bounds.Dy()
	if pw < 1 || ph < 1 || cols < 1 || rows < 1 {
		return
	}
	for row := range rows {
		gy := atR + row
		if gy < 0 || gy >= len(g.screen) {
			continue
		}
		for col := range cols {
			gx := atC + col
			if gx < 0 || gx >= len(g.screen[gy].cells) {
				continue
			}
			x0 := bounds.Min.X + col*pw/cols
			x1 := bounds.Min.X + (col+1)*pw/cols
			y0 := bounds.Min.Y + row*ph/rows
			y1 := bounds.Min.Y + (row+1)*ph/rows
			mid := y0 + (y1-y0)/2
			if mid <= y0 {
				mid = y0 + 1
			}
			if mid > y1 {
				mid = y1
			}
			top, okT := average(img, x0, y0, x1, mid)
			bot, okB := average(img, x0, mid, x1, y1)
			if !okT && !okB {
				continue
			}
			if !okT {
				top = bot
			}
			if !okB {
				bot = top
			}
			g.screen[gy].cells[gx] = glyph{
				r: '▀', fgMode: ColorRGB, fgRGB: top, bgMode: ColorRGB, bgRGB: bot,
			}
		}
	}
}

func average(img image.Image, x0, y0, x1, y1 int) (uint32, bool) {
	if x1 <= x0 || y1 <= y0 {
		return 0, false
	}
	var rs, gs, bs, n uint32
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			rs += uint32(c.R)
			gs += uint32(c.G)
			bs += uint32(c.B)
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return (rs/n)<<16 | (gs/n)<<8 | (bs / n), true
}

func parseKitty(head string) kittyCtl {
	ctl := kittyCtl{action: 't', format: 32, transmission: 'd'}
	for _, part := range strings.Split(head, ",") {
		key, val, ok := strings.Cut(part, "=")
		if !ok || key == "" {
			continue
		}
		switch key {
		case "a":
			if val != "" {
				ctl.action = rune(val[0])
			}
		case "q":
			// quiet is ignored; a bad image is dropped either way
		case "f":
			ctl.format = atoi(val)
		case "t":
			if val != "" {
				ctl.transmission = rune(val[0])
			}
		case "i":
			ctl.id = atoi(val)
		case "m":
			ctl.more = atoi(val)
		case "c":
			ctl.cols = atoi(val)
		case "r":
			ctl.rows = atoi(val)
		case "C":
			ctl.pixW = atoi(val)
		case "R":
			ctl.pixH = atoi(val)
		case "o":
			if val != "" {
				ctl.compression = rune(val[0])
			}
		case "d":
			if val != "" {
				ctl.del = rune(val[0])
			}
		}
	}
	return ctl
}

func mergeKitty(first, last kittyCtl) kittyCtl {
	if last.action == 't' && first.action == 'T' {
		last.action = 'T'
	}
	if first.format != 0 && (last.format == 0 || last.format == 32 && first.format != 32) {
		last.format = first.format
	}
	if last.cols == 0 {
		last.cols = first.cols
	}
	if last.rows == 0 {
		last.rows = first.rows
	}
	if last.pixW == 0 {
		last.pixW = first.pixW
	}
	if last.pixH == 0 {
		last.pixH = first.pixH
	}
	if last.compression == 0 {
		last.compression = first.compression
	}
	if first.transmission != 0 {
		last.transmission = first.transmission
	}
	return last
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func decodeKitty(ctl kittyCtl, payload []byte) image.Image {
	if len(payload) == 0 {
		return nil
	}
	raw, err := decodeB64(payload)
	if err != nil || len(raw) == 0 {
		return nil
	}
	if ctl.compression == 'z' {
		reader, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil
		}
		raw, err = io.ReadAll(io.LimitReader(reader, apcLimit))
		_ = reader.Close()
		if err != nil || len(raw) == 0 {
			return nil
		}
	}
	switch ctl.format {
	case 100:
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return nil
		}
		return img
	case 24:
		return rawImage(raw, ctl.pixW, ctl.pixH, 3)
	case 32:
		return rawImage(raw, ctl.pixW, ctl.pixH, 4)
	default:
		return nil
	}
}

func rawImage(raw []byte, w, h, stride int) image.Image {
	if w < 1 || h < 1 || w > 4096 || h > 4096 || stride < 3 {
		return nil
	}
	need := w * h * stride
	if len(raw) < need {
		return nil
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	i := 0
	for y := range h {
		for x := range w {
			c := color.NRGBA{R: raw[i], G: raw[i+1], B: raw[i+2], A: 255}
			if stride == 4 {
				c.A = raw[i+3]
			}
			img.SetNRGBA(x, y, c)
			i += stride
		}
	}
	return img
}

func decodeB64(payload []byte) ([]byte, error) {
	clean := make([]byte, 0, len(payload))
	for _, b := range payload {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			clean = append(clean, b)
		}
	}
	out, err := base64.StdEncoding.DecodeString(string(clean))
	if err == nil {
		return out, nil
	}
	return base64.RawStdEncoding.DecodeString(string(clean))
}
