package main

import (
	"image"
	"math"
)

// card is the credits-card yellow sampled from the reference gif.
var card = rgb{246, 254, 12}

// paintDuck draws one frame of model into dst. yaw is radians around
// the vertical axis through model.pivot. A mesh with tex is sampled
// from its UVs. dst's bounds start at (0, 0).
func paintDuck(dst *image.RGBA, model mesh, yaw float64) {
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
			row[o] = uint8(card.r)
			row[o+1] = uint8(card.g)
			row[o+2] = uint8(card.b)
			row[o+3] = 255
		}
	}
	// A soft contact shadow sits under the flippers.
	shadow(pix, stride, w, h, w/2, int(float32(h)*0.72), w/5, h/14)

	depth := make([]float32, w*h)
	for i := range depth {
		depth[i] = 1e9
	}
	sign := model.yawSign
	if sign == 0 {
		sign = 1
	}
	yaw32 := float32(yaw)*sign + model.yaw0
	pivot := model.pivot
	eye := vec3{1.65, 0.7, 1.45}
	target := vec3{0, 0.42, 0.12}
	if model.eye != (vec3{}) {
		eye = model.eye
		target = model.aim
	}
	camX, camY, camZ, eye := cameraAt(eye, target)
	light := norm(vec3{-0.35, 0.85, 0.4})
	aspect := float32(w) / float32(h)
	focal := float32(1 / math.Tan(float64(28*math.Pi/180)))
	textured := model.tex != nil

	proj := make([]screenVert, len(model.v))
	for i, v := range model.v {
		p := sub(v.p, pivot)
		p = rotY(p, yaw32)
		p = add(p, pivot)
		n := rotY(v.n, yaw32)
		c := toCam(p, eye, camX, camY, camZ)
		if c.z < 0.05 {
			continue
		}
		sx := (focal/aspect)*(c.x/c.z)*0.5 + 0.5
		sy := 0.5 - focal*(c.y/c.z)*0.5
		proj[i].x = sx * float32(w)
		proj[i].y = sy * float32(h)
		proj[i].z = c.z
		proj[i].invZ = 1 / c.z
		proj[i].u = v.u
		proj[i].v = v.v
		if !textured || v.u < 0 {
			shade := dot(n, light)
			if shade < 0 {
				shade = 0
			}
			gain := 0.38 + 0.62*shade
			proj[i].r = v.c.r * gain
			proj[i].g = v.c.g * gain
			proj[i].b = v.c.b * gain
		}
		proj[i].ok = true
	}
	for _, tri := range model.i {
		a, b, c := proj[tri[0]], proj[tri[1]], proj[tri[2]]
		if !a.ok || !b.ok || !c.ok {
			continue
		}
		fillTri(pix, stride, depth, w, h, model.tex, a, b, c)
	}
}

func camera() (right, up, forward, eye vec3) {
	return cameraAt(vec3{1.65, 0.7, 1.45}, vec3{0, 0.42, 0.12})
}

func cameraAt(eye, target vec3) (right, up, forward, eyeOut vec3) {
	forward = norm(sub(target, eye))
	right = norm(cross(forward, vec3{0, 1, 0}))
	up = cross(right, forward)
	return right, up, forward, eye
}

func toCam(p, eye, right, up, forward vec3) vec3 {
	d := sub(p, eye)
	return vec3{dot(d, right), dot(d, up), dot(d, forward)}
}

type screenVert struct {
	x, y, z float32
	r, g, b float32
	u, v    float32
	invZ    float32
	ok      bool
}

func fillTri(pix []byte, stride int, depth []float32, w, h int, tex *image.RGBA, a, b, c screenVert) {
	area := edge(a.x, a.y, b.x, b.y, c.x, c.y)
	if area > -0.5 && area < 0.5 {
		return
	}
	minX := int(math.Floor(float64(min3(a.x, b.x, c.x))))
	maxX := int(math.Ceil(float64(max3(a.x, b.x, c.x))))
	minY := int(math.Floor(float64(min3(a.y, b.y, c.y))))
	maxY := int(math.Ceil(float64(max3(a.y, b.y, c.y))))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= w {
		maxX = w - 1
	}
	if maxY >= h {
		maxY = h - 1
	}
	inv := 1 / area
	for y := minY; y <= maxY; y++ {
		py := float32(y) + 0.5
		row := pix[y*stride:]
		zb := depth[y*w:]
		for x := minX; x <= maxX; x++ {
			px := float32(x) + 0.5
			w0 := edge(b.x, b.y, c.x, c.y, px, py) * inv
			w1 := edge(c.x, c.y, a.x, a.y, px, py) * inv
			w2 := 1 - w0 - w1
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			z := w0*a.z + w1*b.z + w2*c.z
			if z >= zb[x] {
				continue
			}
			var tr, tg, tb byte
			if tex == nil || a.u < 0 || b.u < 0 || c.u < 0 {
				tr = sat(w0*a.r + w1*b.r + w2*c.r)
				tg = sat(w0*a.g + w1*b.g + w2*c.g)
				tb = sat(w0*a.b + w1*b.b + w2*c.b)
			} else {
				denom := w0*a.invZ + w1*b.invZ + w2*c.invZ
				if denom == 0 {
					continue
				}
				u := (w0*a.u*a.invZ + w1*b.u*b.invZ + w2*c.u*c.invZ) / denom
				v := (w0*a.v*a.invZ + w1*b.v*b.invZ + w2*c.v*c.invZ) / denom
				tr, tg, tb = texel(tex, u, v)
				// Atlas background is pure white. Feather highlights sit just under that.
				if tr > 248 && tg > 248 && tb > 248 {
					continue
				}
			}
			zb[x] = z
			o := x * 4
			row[o] = tr
			row[o+1] = tg
			row[o+2] = tb
			row[o+3] = 255
		}
	}
}

func texel(tex *image.RGBA, u, v float32) (byte, byte, byte) {
	b := tex.Bounds()
	tw, th := b.Dx(), b.Dy()
	if tw < 1 || th < 1 {
		return 0, 0, 0
	}
	if u < 0 {
		u = 0
	} else if u > 1 {
		u = 1
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	x := b.Min.X + int(u*float32(tw-1))
	y := b.Min.Y + int(v*float32(th-1))
	o := tex.PixOffset(x, y)
	p := tex.Pix
	return flipperGreen(p[o], p[o+1], p[o+2])
}

// flipperGreen turns the scan's cyan flippers into the gif's green.
// Sampled flipper pixels on the reference average about rgb(43, 148, 11).
func flipperGreen(r, g, b byte) (byte, byte, byte) {
	if b < 90 || int(b) < int(r)+30 || int(g)+20 < int(r) {
		return r, g, b
	}
	bright := float32(int(g)+int(b)) / (2 * 180)
	if bright > 1 {
		bright = 1
	}
	return sat(24 + 70*bright), sat(80 + 150*bright), sat(6 + 18*bright)
}

func shadow(pix []byte, stride, w, h, cx, cy, rx, ry int) {
	if rx < 1 || ry < 1 {
		return
	}
	for y := cy - ry; y <= cy+ry; y++ {
		if y < 0 || y >= h {
			continue
		}
		for x := cx - rx; x <= cx+rx; x++ {
			if x < 0 || x >= w {
				continue
			}
			dx := float32(x-cx) / float32(rx)
			dy := float32(y-cy) / float32(ry)
			d := dx*dx + dy*dy
			if d > 1 {
				continue
			}
			k := (1 - d) * 0.28
			o := y*stride + x*4
			pix[o] = sat(float32(pix[o]) * (1 - k))
			pix[o+1] = sat(float32(pix[o+1]) * (1 - k))
			pix[o+2] = sat(float32(pix[o+2]) * (1 - k))
		}
	}
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

func sat(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
