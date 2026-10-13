package main

import (
	"image"
	"math"
)

type vec3 struct{ x, y, z float32 }

type rgb struct{ r, g, b float32 }

type vert struct {
	p vec3
	n vec3
	c rgb
	u float32
	v float32
}

type mesh struct {
	v       []vert
	i       [][3]int
	tex     *image.RGBA
	pivot   vec3
	eye     vec3
	aim     vec3
	yaw0    float32
	yawSign float32
}

func add(a, b vec3) vec3         { return vec3{a.x + b.x, a.y + b.y, a.z + b.z} }
func sub(a, b vec3) vec3         { return vec3{a.x - b.x, a.y - b.y, a.z - b.z} }
func mul(a vec3, s float32) vec3 { return vec3{a.x * s, a.y * s, a.z * s} }

func cross(a, b vec3) vec3 {
	return vec3{
		a.y*b.z - a.z*b.y,
		a.z*b.x - a.x*b.z,
		a.x*b.y - a.y*b.x,
	}
}

func dot(a, b vec3) float32 { return a.x*b.x + a.y*b.y + a.z*b.z }

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

func rotX(p vec3, a float32) vec3 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	return vec3{p.x, c*p.y - s*p.z, s*p.y + c*p.z}
}

func (m *mesh) tri(a, b, c vert) {
	n := norm(cross(sub(b.p, a.p), sub(c.p, a.p)))
	if a.n == (vec3{}) {
		a.n, b.n, c.n = n, n, n
	}
	base := len(m.v)
	m.v = append(m.v, a, b, c)
	m.i = append(m.i, [3]int{base, base + 1, base + 2})
}

func (m *mesh) quad(a, b, c, d vec3, col rgb) {
	n := norm(cross(sub(b, a), sub(d, a)))
	va := func(p vec3) vert { return vert{p: p, n: n, c: col} }
	m.tri(va(a), va(b), va(c))
	m.tri(va(a), va(c), va(d))
}

func (m *mesh) ellipsoid(center, radius vec3, latN, lonN int, col rgb) {
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
			a := at(lat, lon)
			b := at(lat, lon+1)
			c := at(lat+1, lon+1)
			d := at(lat+1, lon)
			m.tri(a, d, b)
			m.tri(b, d, c)
		}
	}
}

func (m *mesh) box(center, half vec3, xf func(vec3) vec3, col rgb) {
	if xf == nil {
		xf = func(p vec3) vec3 { return p }
	}
	s := [2]float32{-1, 1}
	corner := func(i, j, k int) vec3 {
		return xf(vec3{
			center.x + s[i]*half.x,
			center.y + s[j]*half.y,
			center.z + s[k]*half.z,
		})
	}
	// i j k as 0/1. Faces listed with an outward corner order in local space.
	p := func(i, j, k int) vec3 { return corner(i, j, k) }
	m.quad(p(0, 0, 0), p(0, 1, 0), p(0, 1, 1), p(0, 0, 1), col)
	m.quad(p(1, 0, 1), p(1, 1, 1), p(1, 1, 0), p(1, 0, 0), col)
	m.quad(p(0, 0, 1), p(1, 0, 1), p(1, 0, 0), p(0, 0, 0), col)
	m.quad(p(0, 1, 0), p(1, 1, 0), p(1, 1, 1), p(0, 1, 1), col)
	m.quad(p(0, 0, 0), p(1, 0, 0), p(1, 1, 0), p(0, 1, 0), col)
	m.quad(p(0, 1, 1), p(1, 1, 1), p(1, 0, 1), p(0, 0, 1), col)
}

// mallard is the example mesh: a low-poly duck in green flippers.
// The Tenor clip this example follows is Animation Factory stock footage
// of that pose, composited on a Cyberpunk 2077 credits card. That mesh
// was never released, so the triangles here are original.
func mallard() mesh {
	m := mesh{pivot: vec3{0, 0.45, 0.1}}
	body := rgb{232, 196, 48}
	belly := rgb{246, 220, 90}
	head := rgb{18, 92, 42}
	chest := rgb{28, 32, 28}
	beak := rgb{232, 122, 28}
	fin := rgb{36, 214, 72}
	eye := rgb{12, 12, 12}

	m.ellipsoid(vec3{0, 0.42, 0.02}, vec3{0.40, 0.30, 0.58}, 10, 16, body)
	m.ellipsoid(vec3{0, 0.30, 0.05}, vec3{0.30, 0.16, 0.42}, 6, 12, belly)
	m.ellipsoid(vec3{0, 0.58, 0.42}, vec3{0.22, 0.20, 0.18}, 6, 10, chest)
	m.ellipsoid(vec3{0, 0.68, 0.46}, vec3{0.12, 0.16, 0.13}, 6, 8, head)
	m.ellipsoid(vec3{0, 0.92, 0.62}, vec3{0.20, 0.17, 0.22}, 8, 12, head)
	m.ellipsoid(vec3{0, 0.90, 0.88}, vec3{0.08, 0.055, 0.16}, 5, 8, beak)
	m.ellipsoid(vec3{0.08, 0.97, 0.74}, vec3{0.032, 0.032, 0.032}, 4, 6, eye)
	m.ellipsoid(vec3{-0.08, 0.97, 0.74}, vec3{0.032, 0.032, 0.032}, 4, 6, eye)
	m.ellipsoid(vec3{0, 0.52, -0.58}, vec3{0.09, 0.07, 0.20}, 4, 6, body)

	flipper := func(side float32) {
		xf := func(p vec3) vec3 {
			p = rotY(p, side*0.45)
			p = rotX(p, -0.2)
			return add(p, vec3{side * 0.10, 0.05, 0.02})
		}
		m.box(vec3{0, 0, 0.22}, vec3{0.15, 0.016, 0.34}, xf, fin)
	}
	flipper(-1)
	flipper(1)
	return m
}
