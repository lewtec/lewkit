package main

import (
	"image"
	"math"

	"github.com/lewtec/lewkit/x/ui/world"
)

// The tutorial's chess kit is a Sketchfab model. These pieces are original.
// The camera matches the article's setup: high, off the board's side,
// looking at the center. It backs up until the board fits the frame.
// A tall frame keeps the horizontal field and shows more above and below.
// Squares are the article's pale and near-black planes. A warm key and a
// cool fill shade the pieces. Square tops stay evenly lit.

const squareTop float32 = 0.08

var (
	colTable    = rgb{18, 22, 20}
	colPlinth   = rgb{86, 52, 30}
	colLight    = rgb{255, 230, 230}
	colDark     = rgb{0, 26, 26}
	colHover    = rgb{204, 77, 77}
	colSelected = rgb{230, 26, 26}
	colIvory    = rgb{248, 232, 186}
	colEbony    = rgb{54, 44, 36}
)

type vec3 struct{ x, y, z float32 }

type rgb struct{ r, g, b float32 }

type vert struct {
	p vec3
	n vec3
	c rgb
}

type model struct {
	v []vert
	i [][3]int
}

func add(a, b vec3) vec3         { return vec3{a.x + b.x, a.y + b.y, a.z + b.z} }
func sub(a, b vec3) vec3         { return vec3{a.x - b.x, a.y - b.y, a.z - b.z} }
func mul(a vec3, s float32) vec3 { return vec3{a.x * s, a.y * s, a.z * s} }
func dot(a, b vec3) float32      { return a.x*b.x + a.y*b.y + a.z*b.z }

func cross(a, b vec3) vec3 {
	return vec3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}

func norm(a vec3) vec3 {
	l := float32(math.Sqrt(float64(dot(a, a))))
	if l < 1e-8 {
		return vec3{}
	}
	return mul(a, 1/l)
}

func rotY(p vec3, yaw float32) vec3 {
	c, s := float32(math.Cos(float64(yaw))), float32(math.Sin(float64(yaw)))
	return vec3{c*p.x + s*p.z, p.y, -s*p.x + c*p.z}
}

func (m *model) tri(a, b, c vert) {
	if a.n == (vec3{}) {
		n := norm(cross(sub(b.p, a.p), sub(c.p, a.p)))
		a.n, b.n, c.n = n, n, n
	}
	base := len(m.v)
	m.v = append(m.v, a, b, c)
	m.i = append(m.i, [3]int{base, base + 1, base + 2})
}

func (m *model) quad(a, b, c, d vec3, col rgb) {
	n := norm(cross(sub(b, a), sub(d, a)))
	at := func(p vec3) vert { return vert{p: p, n: n, c: col} }
	m.tri(at(a), at(b), at(c))
	m.tri(at(a), at(c), at(d))
}

func (m *model) box(center, half vec3, yaw float32, col rgb) {
	s := [2]float32{-1, 1}
	corner := func(i, j, k int) vec3 {
		p := vec3{center.x + s[i]*half.x, center.y + s[j]*half.y, center.z + s[k]*half.z}
		return rotY(p, yaw)
	}
	p := func(i, j, k int) vec3 { return corner(i, j, k) }
	// Winding puts the normal on the outside, toward the light.
	m.quad(p(0, 0, 0), p(0, 0, 1), p(0, 1, 1), p(0, 1, 0), col)
	m.quad(p(1, 0, 1), p(1, 0, 0), p(1, 1, 0), p(1, 1, 1), col)
	m.quad(p(0, 0, 1), p(0, 0, 0), p(1, 0, 0), p(1, 0, 1), col)
	m.quad(p(0, 1, 0), p(0, 1, 1), p(1, 1, 1), p(1, 1, 0), col)
	m.quad(p(0, 0, 0), p(0, 1, 0), p(1, 1, 0), p(1, 0, 0), col)
	m.quad(p(0, 1, 1), p(0, 0, 1), p(1, 0, 1), p(1, 1, 1), col)
}

func (m *model) ellipsoid(center, radius vec3, latN, lonN int, col rgb) {
	if latN < 2 || lonN < 3 {
		return
	}
	at := func(lat, lon int) vert {
		v := float64(lat) / float64(latN)
		u := float64(lon) / float64(lonN)
		phi := (v - 0.5) * math.Pi
		theta := u * 2 * math.Pi
		nx := float32(math.Cos(phi) * math.Cos(theta))
		ny := float32(math.Sin(phi))
		nz := float32(math.Cos(phi) * math.Sin(theta))
		return vert{
			p: vec3{center.x + nx*radius.x, center.y + ny*radius.y, center.z + nz*radius.z},
			n: norm(vec3{nx / radius.x, ny / radius.y, nz / radius.z}),
			c: col,
		}
	}
	for lat := 0; lat < latN; lat++ {
		for lon := 0; lon < lonN; lon++ {
			a, b := at(lat, lon), at(lat, lon+1)
			c, d := at(lat+1, lon+1), at(lat+1, lon)
			m.tri(a, d, b)
			m.tri(b, d, c)
		}
	}
}

// lathe spins a (radius, height) profile around Y. The profile runs
// from the base to the crown.
func (m *model) lathe(profile []vec3, sides int, col rgb) {
	if sides < 3 || len(profile) < 2 {
		return
	}
	rings := make([][]vert, len(profile))
	for i, p := range profile {
		prev, next := p, p
		if i > 0 {
			prev = profile[i-1]
		}
		if i+1 < len(profile) {
			next = profile[i+1]
		}
		dy, dr := next.y-prev.y, next.x-prev.x
		nr, ny := norm(vec3{dy, -dr, 0}).x, norm(vec3{dy, -dr, 0}).y
		ring := make([]vert, sides)
		for s := 0; s < sides; s++ {
			theta := 2 * math.Pi * float64(s) / float64(sides)
			ct, st := float32(math.Cos(theta)), float32(math.Sin(theta))
			ring[s] = vert{
				p: vec3{p.x * ct, p.y, p.x * st},
				n: norm(vec3{nr * ct, ny, nr * st}),
				c: col,
			}
		}
		rings[i] = ring
	}
	for i := 0; i+1 < len(rings); i++ {
		for s := 0; s < sides; s++ {
			s2 := (s + 1) % sides
			a, b := rings[i][s], rings[i][s2]
			d, c := rings[i+1][s], rings[i+1][s2]
			m.tri(a, d, b)
			m.tri(b, d, c)
		}
	}
}

func white() rgb { return rgb{1, 1, 1} }

func kingMesh() model {
	var m model
	m.lathe([]vec3{
		{0, 0, 0}, {0.32, 0, 0}, {0.34, 0.08, 0}, {0.22, 0.16, 0},
		{0.16, 0.55, 0}, {0.12, 0.78, 0}, {0.18, 0.88, 0}, {0.08, 1.02, 0}, {0, 1.08, 0},
	}, 12, white())
	m.box(vec3{0, 1.18, 0}, vec3{0.035, 0.12, 0.035}, 0, white())
	m.box(vec3{0, 1.22, 0}, vec3{0.11, 0.03, 0.03}, 0, white())
	return m
}

func queenMesh() model {
	var m model
	m.lathe([]vec3{
		{0, 0, 0}, {0.32, 0, 0}, {0.34, 0.08, 0}, {0.20, 0.16, 0},
		{0.15, 0.62, 0}, {0.11, 0.82, 0}, {0.20, 0.92, 0}, {0.06, 1.08, 0}, {0, 1.16, 0},
	}, 12, white())
	return m
}

func bishopMesh() model {
	var m model
	m.lathe([]vec3{
		{0, 0, 0}, {0.30, 0, 0}, {0.32, 0.08, 0}, {0.18, 0.16, 0},
		{0.14, 0.58, 0}, {0.10, 0.78, 0}, {0.13, 0.88, 0}, {0.04, 1.02, 0}, {0, 1.08, 0},
	}, 12, white())
	return m
}

func knightMesh() model {
	var m model
	m.lathe([]vec3{
		{0, 0, 0}, {0.30, 0, 0}, {0.32, 0.07, 0}, {0.20, 0.12, 0}, {0, 0.16, 0},
	}, 12, white())
	m.ellipsoid(vec3{0.02, 0.30, 0}, vec3{0.16, 0.14, 0.13}, 6, 10, white())
	m.ellipsoid(vec3{0.12, 0.46, 0}, vec3{0.10, 0.16, 0.09}, 6, 8, white())
	m.ellipsoid(vec3{0.26, 0.64, 0}, vec3{0.14, 0.11, 0.09}, 6, 8, white())
	m.ellipsoid(vec3{0.40, 0.58, 0}, vec3{0.12, 0.06, 0.055}, 5, 8, white())
	m.ellipsoid(vec3{0.22, 0.76, 0.05}, vec3{0.035, 0.07, 0.03}, 4, 6, white())
	m.ellipsoid(vec3{0.22, 0.76, -0.05}, vec3{0.035, 0.07, 0.03}, 4, 6, white())
	return m
}

func rookMesh() model {
	var m model
	m.lathe([]vec3{
		{0, 0, 0}, {0.32, 0, 0}, {0.34, 0.08, 0}, {0.22, 0.14, 0},
		{0.20, 0.62, 0}, {0.28, 0.70, 0}, {0.28, 0.78, 0}, {0.16, 0.78, 0}, {0, 0.82, 0},
	}, 12, white())
	for i := range 4 {
		m.box(vec3{0.20, 0.90, 0}, vec3{0.07, 0.07, 0.07}, float32(i)*math.Pi/2, white())
	}
	return m
}

func pawnMesh() model {
	var m model
	m.lathe([]vec3{
		{0, 0, 0}, {0.26, 0, 0}, {0.28, 0.07, 0}, {0.16, 0.14, 0},
		{0.11, 0.36, 0}, {0.13, 0.46, 0}, {0.18, 0.56, 0}, {0.10, 0.66, 0}, {0, 0.70, 0},
	}, 10, white())
	return m
}

// squareMesh is the article's 1×1 plane. The winding points up.
func squareMesh() model {
	var m model
	y := squareTop
	m.quad(
		vec3{-0.5, y, -0.5},
		vec3{-0.5, y, 0.5},
		vec3{0.5, y, 0.5},
		vec3{0.5, y, -0.5},
		white(),
	)
	return m
}

func plinthMesh() model {
	var m model
	m.box(vec3{3.5, -0.06, 3.5}, vec3{4.35, 0.08, 4.35}, 0, white())
	return m
}

var (
	pieceModels = [...]model{kingMesh(), queenMesh(), bishopMesh(), knightMesh(), rookMesh(), pawnMesh()}
	oneSquare   = squareMesh()
	onePlinth   = plinthMesh()
)

type camera struct {
	eye, right, up, forward vec3
	scaleX, scaleY          float32
	w, h                    int
}

func lookAt(size image.Point) camera {
	w, h := size.X, size.Y
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	// The article puts the camera at (-7, 20, 4). The eye stays on that
	// side and moves along the same ray until the board fits.
	target := vec3{3.5, 0, 3.5}
	approach := norm(sub(vec3{-5.5, 10.5, 0.2}, target))
	fov := float32(34 * math.Pi / 180)
	focal := float32(1 / math.Tan(float64(fov/2)))
	aspect := float32(w) / float32(h)
	// The shorter edge keeps the 34° field. The longer edge opens, so a
	// tall frame does not crop the sides of the board.
	scaleX, scaleY := focal/aspect, focal
	if aspect < 1 {
		scaleX, scaleY = focal, focal*aspect
	}
	top, rim := framePad(w, h)
	lo, hi := float32(6), float32(70)
	for range 20 {
		dist := (lo + hi) * 0.5
		cam := placeCamera(target, approach, dist, scaleX, scaleY, w, h)
		if boardInside(cam, top, rim) {
			hi = dist
			continue
		}
		lo = dist
	}
	return placeCamera(target, approach, hi, scaleX, scaleY, w, h)
}

func placeCamera(target, approach vec3, dist, scaleX, scaleY float32, w, h int) camera {
	eye := add(target, mul(approach, dist))
	forward := norm(sub(target, eye))
	right := norm(cross(forward, vec3{0, 1, 0}))
	up := cross(right, forward)
	return camera{
		eye: eye, right: right, up: up, forward: forward,
		scaleX: scaleX, scaleY: scaleY,
		w: w, h: h,
	}
}

// framePad is the clear band above the board, for the move line, and the
// rim kept around the other three edges.
func framePad(w, h int) (top, rim float32) {
	top, rim = 36, 12
	if h < 80 || w < 80 {
		top, rim = 8, 4
	}
	return top, rim
}

// boardHull is the plinth and the tallest piece, in world space.
func boardHull() []vec3 {
	var pts []vec3
	for _, x := range []float32{-0.9, 8.05} {
		for _, y := range []float32{-0.14, 1.45} {
			for _, z := range []float32{-0.9, 8.05} {
				pts = append(pts, vec3{x, y, z})
			}
		}
	}
	return pts
}

func boardInside(cam camera, top, rim float32) bool {
	limitX := float32(cam.w) - rim
	limitY := float32(cam.h) - rim
	for _, p := range boardHull() {
		x, y, _, ok := cam.project(p)
		if !ok || x < rim || x > limitX || y < top || y > limitY {
			return false
		}
	}
	return true
}

func (c camera) project(p vec3) (x, y, z float32, ok bool) {
	d := sub(p, c.eye)
	q := vec3{dot(d, c.right), dot(d, c.up), dot(d, c.forward)}
	if q.z < 0.05 {
		return 0, 0, 0, false
	}
	rx, ry := q.x/q.z, q.y/q.z
	x = (c.scaleX*rx*0.5 + 0.5) * float32(c.w)
	y = (0.5 - c.scaleY*ry*0.5) * float32(c.h)
	return x, y, q.z, true
}

func (c camera) ray(px, py float32) (origin, dir vec3) {
	sx := px / float32(c.w)
	sy := py / float32(c.h)
	cx := (sx - 0.5) * 2 / c.scaleX
	cy := (0.5 - sy) * 2 / c.scaleY
	dir = norm(add(add(mul(c.right, cx), mul(c.up, cy)), c.forward))
	return c.eye, dir
}

// squareCenter is the pixel at the middle of rank x, file y.
func squareCenter(size image.Point, x, y uint8) image.Point {
	c := lookAt(size)
	sx, sy, _, ok := c.project(vec3{float32(x) + 0.5, squareTop, float32(y) + 0.5})
	if !ok {
		return image.Point{}
	}
	return image.Pt(int(sx), int(sy))
}

// pickSquare is the square under a window pixel. The ray hits the
// square tops, which is the tutorial's board pick without a mesh ray.
func pickSquare(size image.Point, at image.Point) (uint8, uint8, bool) {
	c := lookAt(size)
	origin, dir := c.ray(float32(at.X)+0.5, float32(at.Y)+0.5)
	if dir.y > -1e-4 {
		return 0, 0, false
	}
	t := (squareTop - origin.y) / dir.y
	if t < 0 {
		return 0, 0, false
	}
	hit := add(origin, mul(dir, t))
	if hit.x < 0 || hit.z < 0 || hit.x >= 8 || hit.z >= 8 {
		return 0, 0, false
	}
	return uint8(hit.x), uint8(hit.z), true
}

type screenVert struct {
	x, y, z    float32
	r, g, b    float32
	nx, ny, nz float32
	wx, wy, wz float32
	ok         bool
}

// pixelBox is the screen span of one square, so a hover can recolor
// that square without walking the whole frame.
type pixelBox struct {
	minX, minY int
	maxX, maxY int
}

func (b pixelBox) empty() bool { return b.maxX < b.minX }

func (b *pixelBox) add(x, y int) {
	if b.empty() {
		b.minX, b.maxX = x, x
		b.minY, b.maxY = y, y
		return
	}
	if x < b.minX {
		b.minX = x
	}
	if x > b.maxX {
		b.maxX = x
	}
	if y < b.minY {
		b.minY = y
	}
	if y > b.maxY {
		b.maxY = y
	}
}

// squareLit is the gain on a square's upward face.
func squareLit() float32 {
	light := norm(vec3{0.25, 1, 0.2})
	if light.y < 0 {
		return 0.46
	}
	return 0.46 + 0.54*light.y
}

// lit shades one face. An upward normal keeps squareLit, so the board
// stays pale and near-black. A side takes a warm key from the camera's
// left, a cool fill from the right, and a tight highlight.
func lit(n, world, eye vec3) (r, g, b float32) {
	up := n.y
	if up < 0 {
		up = 0
	}
	flat := squareLit()
	key := norm(vec3{-0.85, 0.72, 0.42})
	fill := norm(vec3{0.62, 0.22, -0.72})
	kd := dot(n, key)
	if kd < 0 {
		kd = 0
	}
	fd := dot(n, fill)
	if fd < 0 {
		fd = 0
	}
	side := 1 - up*up
	sr := 0.18 + 1.15*kd + 0.10*fd
	sg := 0.16 + 0.98*kd + 0.14*fd
	sb := 0.14 + 0.72*kd + 0.32*fd
	view := norm(sub(eye, world))
	half := norm(add(key, view))
	spec := dot(n, half)
	if spec < 0 {
		spec = 0
	}
	spec = spec * spec
	spec = spec * spec
	shine := spec * spec * 0.28 * side
	r = flat*up + sr*side + shine
	g = flat*up + sg*side + shine
	b = flat*up + sb*side + shine*0.85
	return r, g, b
}

type canvas struct {
	pix    []byte
	stride int
	depth  []float32
	cover  []uint8
	boxes  *[64]pixelBox
	proj   []screenVert
	w, h   int
	mark   uint8
}

// paintChess draws the board and the pieces into dst. cover receives
// the square under each pixel (0 is empty, otherwise rank*8+file+1)
// so a later hover can recolor those pixels. Piece positions are the
// sliding poses.
func paintChess(dst *image.RGBA, sim *world.Sim, depth []float32, cover []uint8, boxes *[64]pixelBox) {
	if dst == nil {
		return
	}
	b := dst.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 2 || h < 2 {
		return
	}
	pix := dst.Pix
	stride := dst.Stride
	for y := 0; y < h; y++ {
		row := pix[y*stride : y*stride+w*4]
		for x := 0; x < w; x++ {
			o := x * 4
			row[o] = uint8(colTable.r)
			row[o+1] = uint8(colTable.g)
			row[o+2] = uint8(colTable.b)
			row[o+3] = 255
		}
	}
	if len(depth) < w*h {
		return
	}
	for i := range w * h {
		depth[i] = 1e9
	}
	if cover != nil {
		clear(cover[:w*h])
	}
	if boxes != nil {
		for i := range boxes {
			boxes[i] = pixelBox{maxX: -1}
		}
	}
	cv := canvas{pix: pix, stride: stride, depth: depth, cover: cover, boxes: boxes, w: w, h: h}
	cam := lookAt(image.Pt(w, h))
	cv.draw(cam, onePlinth, 0, vec3{}, colPlinth)
	if sim == nil || sim.World == nil {
		return
	}
	shades := world.Query[shade](sim.World)
	world.Query[square](sim.World).Read(func(e world.Entity, sq square) {
		tint := colDark
		if sq.light() {
			tint = colLight
		}
		if got, ok := shades.Get(e); ok {
			switch got {
			case shadeHover:
				tint = colHover
			case shadeSelected:
				tint = colSelected
			case shadeLight:
				tint = colLight
			default:
				tint = colDark
			}
		}
		cv.mark = 0
		if sq.x < 8 && sq.y < 8 {
			cv.mark = sq.x*8 + sq.y + 1
		}
		origin := vec3{float32(sq.x) + 0.5, 0, float32(sq.y) + 0.5}
		cv.draw(cam, oneSquare, 0, origin, tint)
	})
	cv.mark = 0
	world.Query[piece](sim.World).Read(func(e world.Entity, body piece) {
		at := pose{x: float32(body.x), z: float32(body.y)}
		if got, ok := world.Query[pose](sim.World).Get(e); ok {
			at = got
		}
		tint := colIvory
		yaw := float32(0)
		if body.color == sideBlack {
			tint = colEbony
			yaw = math.Pi
		}
		origin := vec3{at.x + 0.5, squareTop, at.z + 0.5}
		mesh := pieceModels[0]
		if int(body.kind) < len(pieceModels) {
			mesh = pieceModels[body.kind]
		}
		cv.draw(cam, mesh, yaw, origin, tint)
	})
}

func (cv *canvas) draw(cam camera, mesh model, yaw float32, origin vec3, tint rgb) {
	if cap(cv.proj) < len(mesh.v) {
		cv.proj = make([]screenVert, len(mesh.v))
	}
	proj := cv.proj[:len(mesh.v)]
	for i, v := range mesh.v {
		p := add(rotY(v.p, yaw), origin)
		n := rotY(v.n, yaw)
		sx, sy, z, ok := cam.project(p)
		if !ok {
			proj[i] = screenVert{}
			continue
		}
		lr, lg, lb := lit(n, p, cam.eye)
		proj[i] = screenVert{
			x: sx, y: sy, z: z, ok: true,
			r:  tint.r * v.c.r * lr,
			g:  tint.g * v.c.g * lg,
			b:  tint.b * v.c.b * lb,
			nx: n.x, ny: n.y, nz: n.z,
			wx: p.x, wy: p.y, wz: p.z,
		}
	}
	for _, tri := range mesh.i {
		a, b, c := proj[tri[0]], proj[tri[1]], proj[tri[2]]
		if !a.ok || !b.ok || !c.ok || !faces(cam, a, b, c) {
			continue
		}
		cv.fill(a, b, c)
	}
}

// faces reports whether the triangle's outside points toward the camera.
func faces(cam camera, a, b, c screenVert) bool {
	n := vec3{a.nx + b.nx + c.nx, a.ny + b.ny + c.ny, a.nz + b.nz + c.nz}
	mid := vec3{(a.wx + b.wx + c.wx) / 3, (a.wy + b.wy + c.wy) / 3, (a.wz + b.wz + c.wz) / 3}
	return dot(n, sub(cam.eye, mid)) > 0
}

func (cv *canvas) fill(a, b, c screenVert) {
	area := edge(a.x, a.y, b.x, b.y, c.x, c.y)
	if area > -0.5 && area < 0.5 {
		return
	}
	minY := int(math.Floor(float64(min3(a.y, b.y, c.y))))
	maxY := int(math.Ceil(float64(max3(a.y, b.y, c.y))))
	if minY < 0 {
		minY = 0
	}
	if maxY >= cv.h {
		maxY = cv.h - 1
	}
	boxL := int(math.Floor(float64(min3(a.x, b.x, c.x))))
	boxR := int(math.Ceil(float64(max3(a.x, b.x, c.x))))
	inv := 1 / area
	dw0 := (c.y - b.y) * inv
	dw1 := (a.y - c.y) * inv
	dw2 := (b.y - a.y) * inv
	dz := dw0*a.z + dw1*b.z + dw2*c.z
	dr := dw0*a.r + dw1*b.r + dw2*c.r
	dg := dw0*a.g + dw1*b.g + dw2*c.g
	db := dw0*a.b + dw1*b.b + dw2*c.b
	for y := minY; y <= maxY; y++ {
		py := float32(y) + 0.5
		minX, maxX := boxL, boxR
		if lo, hi, ok := rowSpan(a, b, c, py); ok {
			minX = int(math.Floor(float64(lo))) - 1
			maxX = int(math.Ceil(float64(hi))) + 1
		}
		if minX < 0 {
			minX = 0
		}
		if maxX >= cv.w {
			maxX = cv.w - 1
		}
		if minX > maxX {
			continue
		}
		px := float32(minX) + 0.5
		w0 := edge(b.x, b.y, c.x, c.y, px, py) * inv
		w1 := edge(c.x, c.y, a.x, a.y, px, py) * inv
		w2 := 1 - w0 - w1
		z := w0*a.z + w1*b.z + w2*c.z
		r := w0*a.r + w1*b.r + w2*c.r
		g := w0*a.g + w1*b.g + w2*c.g
		bl := w0*a.b + w1*b.b + w2*c.b
		row := cv.pix[y*cv.stride:]
		zb := cv.depth[y*cv.w:]
		var cov []uint8
		if cv.cover != nil {
			cov = cv.cover[y*cv.w:]
		}
		for x := minX; x <= maxX; x++ {
			if w0 >= 0 && w1 >= 0 && w2 >= 0 && z < zb[x] {
				zb[x] = z
				o := x * 4
				row[o] = sat(r)
				row[o+1] = sat(g)
				row[o+2] = sat(bl)
				row[o+3] = 255
				if cov != nil {
					cov[x] = cv.mark
					if cv.mark != 0 && cv.boxes != nil {
						cv.boxes[cv.mark-1].add(x, y)
					}
				}
			}
			w0 += dw0
			w1 += dw1
			w2 += dw2
			z += dz
			r += dr
			g += dg
			bl += db
		}
	}
}

func rowSpan(a, b, c screenVert, py float32) (float32, float32, bool) {
	lo, hi := float32(0), float32(0)
	n := 0
	hit := func(p, q screenVert) {
		if (py < p.y && py < q.y) || (py > p.y && py > q.y) || p.y == q.y {
			return
		}
		t := (py - p.y) / (q.y - p.y)
		if t < 0 || t > 1 {
			return
		}
		x := p.x + t*(q.x-p.x)
		if n == 0 || x < lo {
			lo = x
		}
		if n == 0 || x > hi {
			hi = x
		}
		n++
	}
	hit(a, b)
	hit(b, c)
	hit(c, a)
	return lo, hi, n > 0
}

func edge(ax, ay, bx, by, px, py float32) float32 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

func min3(a, b, c float32) float32 {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func max3(a, b, c float32) float32 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func sat(v float32) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}
