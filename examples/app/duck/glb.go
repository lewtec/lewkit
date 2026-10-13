package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"slices"
	"sync"
)

//go:embed embed/pato.glb
var patoGLB []byte

// pato is the photogrammetry scan of the gif mallard.
// The file is one mesh with positions, UVs, and a base-color image.
// Normal and occlusion maps in the file are left unused: the painter samples the photo as-is.
func pato() (mesh, error) {
	patoOnce.Do(func() {
		patoMesh, patoErr = loadGLB(patoGLB)
	})
	return patoMesh, patoErr
}

var (
	patoOnce sync.Once
	patoMesh mesh
	patoErr  error
)

type gltfDoc struct {
	Accessors   []gltfAccessor `json:"accessors"`
	BufferViews []gltfView     `json:"bufferViews"`
	Meshes      []gltfMesh     `json:"meshes"`
	Materials   []gltfMaterial `json:"materials"`
	Textures    []gltfTexture  `json:"textures"`
	Images      []gltfImage    `json:"images"`
}

type gltfAccessor struct {
	BufferView    int    `json:"bufferView"`
	ByteOffset    int    `json:"byteOffset"`
	ComponentType int    `json:"componentType"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
}

type gltfView struct {
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
	ByteStride int `json:"byteStride"`
}

type gltfMesh struct {
	Primitives []gltfPrim `json:"primitives"`
}

type gltfPrim struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices"`
	Material   *int           `json:"material"`
}

type gltfMaterial struct {
	PBR *struct {
		BaseColorTexture *struct {
			Index int `json:"index"`
		} `json:"baseColorTexture"`
	} `json:"pbrMetallicRoughness"`
}

type gltfTexture struct {
	Source int `json:"source"`
}

type gltfImage struct {
	MimeType   string `json:"mimeType"`
	BufferView int    `json:"bufferView"`
}

func loadGLB(src []byte) (mesh, error) {
	if len(src) < 20 {
		return mesh{}, fmt.Errorf("pato: short glb")
	}
	magic, _, length := u32(src[0:4]), u32(src[4:8]), u32(src[8:12])
	if magic != 0x46546C67 || int(length) > len(src) {
		return mesh{}, fmt.Errorf("pato: not a glb")
	}
	var doc gltfDoc
	var bin []byte
	off := 12
	for off+8 <= len(src) && off < int(length) {
		clen := int(u32(src[off : off+4]))
		ctype := u32(src[off+4 : off+8])
		off += 8
		if clen < 0 || off+clen > len(src) {
			return mesh{}, fmt.Errorf("pato: bad chunk")
		}
		chunk := src[off : off+clen]
		off += clen
		switch ctype {
		case 0x4E4F534A:
			if err := json.Unmarshal(chunk, &doc); err != nil {
				return mesh{}, fmt.Errorf("pato: %w", err)
			}
		case 0x004E4942:
			bin = chunk
		}
	}
	if bin == nil || len(doc.Meshes) == 0 {
		return mesh{}, fmt.Errorf("pato: empty glb")
	}
	var out mesh
	for _, m := range doc.Meshes {
		for _, prim := range m.Primitives {
			if err := appendPrim(&out, doc, bin, prim); err != nil {
				return mesh{}, err
			}
		}
	}
	if len(out.i) == 0 || out.tex == nil {
		return mesh{}, fmt.Errorf("pato: mesh has no textured triangles")
	}
	compact(&out)
	dropFins(&out)
	keepBody(&out)
	stage(&out)
	trimTail(&out)
	stage(&out)
	addBlades(&out)
	return out, nil
}

// keepBody drops the small islands of slab that stay attached to nothing.
func keepBody(m *mesh) {
	if len(m.i) == 0 || len(m.v) == 0 {
		return
	}
	parent := make([]int, len(m.v))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(a int) int {
		for parent[a] != a {
			parent[a] = parent[parent[a]]
			a = parent[a]
		}
		return a
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for _, tri := range m.i {
		union(tri[0], tri[1])
		union(tri[1], tri[2])
	}
	counts := map[int]int{}
	best, bestN := 0, 0
	for _, tri := range m.i {
		root := find(tri[0])
		counts[root]++
		if counts[root] > bestN {
			best, bestN = root, counts[root]
		}
	}
	kept := m.i[:0]
	for _, tri := range m.i {
		root := find(tri[0])
		if root == best || counts[root]*8 > bestN {
			kept = append(kept, tri)
		}
	}
	m.i = kept
	compact(m)
}

// trimTail removes a small island left under the feet by the stand cut.
func trimTail(m *mesh) {
	if len(m.v) < 8 || len(m.i) == 0 {
		return
	}
	ys := make([]float32, len(m.v))
	for i, v := range m.v {
		ys[i] = v.p.y
	}
	slices.Sort(ys)
	cut := float32(0)
	found := false
	for i := 1; i < len(ys); i++ {
		if ys[i]-ys[i-1] > 0.06 && i < len(ys)/8 {
			cut = ys[i-1] + 0.001
			found = true
			break
		}
	}
	if !found {
		return
	}
	kept := m.i[:0]
	for _, tri := range m.i {
		if m.v[tri[0]].p.y <= cut && m.v[tri[1]].p.y <= cut && m.v[tri[2]].p.y <= cut {
			continue
		}
		kept = append(kept, tri)
	}
	m.i = kept
	compact(m)
}

// compact drops vertices that only belonged to the removed stand.
func compact(m *mesh) {
	used := make([]int, len(m.v))
	for i := range used {
		used[i] = -1
	}
	next := 0
	verts := make([]vert, 0, len(m.v))
	for t := range m.i {
		for k := 0; k < 3; k++ {
			id := m.i[t][k]
			if used[id] < 0 {
				used[id] = next
				verts = append(verts, m.v[id])
				next++
			}
			m.i[t][k] = used[id]
		}
	}
	m.v = verts
}

func appendPrim(out *mesh, doc gltfDoc, bin []byte, prim gltfPrim) error {
	posAt, ok := prim.Attributes["POSITION"]
	if !ok || prim.Indices == nil {
		return fmt.Errorf("pato: primitive has no positions")
	}
	uvAt, ok := prim.Attributes["TEXCOORD_0"]
	if !ok {
		return fmt.Errorf("pato: primitive has no uvs")
	}
	pos, err := readVec(bin, doc, posAt, 3)
	if err != nil {
		return err
	}
	uv, err := readVec(bin, doc, uvAt, 2)
	if err != nil {
		return err
	}
	if len(uv) != len(pos) {
		return fmt.Errorf("pato: uv count %d != positions %d", len(uv), len(pos))
	}
	idx, err := readIndex(bin, doc, *prim.Indices)
	if err != nil {
		return err
	}
	if len(idx)%3 != 0 {
		return fmt.Errorf("pato: index count %d", len(idx))
	}
	tex, err := baseColor(doc, bin, prim.Material)
	if err != nil {
		return err
	}
	if out.tex == nil {
		out.tex = tex
	}
	base := len(out.v)
	for i, p := range pos {
		out.v = append(out.v, vert{p: p, u: uv[i].x, v: uv[i].y})
	}
	for i := 0; i < len(idx); i += 3 {
		a, b, c := idx[i], idx[i+1], idx[i+2]
		if a < 0 || b < 0 || c < 0 || a >= len(pos) || b >= len(pos) || c >= len(pos) {
			return fmt.Errorf("pato: index out of range")
		}
		// The capture includes the white slab the duck was standing on.
		// Flippers are saturated, so a pale triangle under the body is the slab.
		top := pos[a].y
		if pos[b].y > top {
			top = pos[b].y
		}
		if pos[c].y > top {
			top = pos[c].y
		}
		whites := 0
		if stand(tex, uv[a].x, uv[a].y) {
			whites++
		}
		if stand(tex, uv[b].x, uv[b].y) {
			whites++
		}
		if stand(tex, uv[c].x, uv[c].y) {
			whites++
		}
		if top < 0.50 && whites == 3 {
			continue
		}
		out.i = append(out.i, [3]int{base + a, base + b, base + c})
	}
	return nil
}

// stand is the white slab under the duck. Saturated texels are the flippers.
func stand(tex *image.RGBA, u, v float32) bool {
	r, g, b := texel(tex, u, v)
	max := r
	min := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}
	return min > 205 && int(max)-int(min) < 30
}

func baseColor(doc gltfDoc, bin []byte, material *int) (*image.RGBA, error) {
	if material == nil || *material < 0 || *material >= len(doc.Materials) {
		return nil, fmt.Errorf("pato: no material")
	}
	pbr := doc.Materials[*material].PBR
	if pbr == nil || pbr.BaseColorTexture == nil {
		return nil, fmt.Errorf("pato: no base color")
	}
	ti := pbr.BaseColorTexture.Index
	if ti < 0 || ti >= len(doc.Textures) {
		return nil, fmt.Errorf("pato: bad texture")
	}
	si := doc.Textures[ti].Source
	if si < 0 || si >= len(doc.Images) {
		return nil, fmt.Errorf("pato: bad image")
	}
	view := doc.Images[si].BufferView
	if view < 0 || view >= len(doc.BufferViews) {
		return nil, fmt.Errorf("pato: bad image view")
	}
	bv := doc.BufferViews[view]
	if bv.ByteOffset < 0 || bv.ByteLength < 1 || bv.ByteOffset+bv.ByteLength > len(bin) {
		return nil, fmt.Errorf("pato: image out of range")
	}
	src, _, err := image.Decode(bytes.NewReader(bin[bv.ByteOffset : bv.ByteOffset+bv.ByteLength]))
	if err != nil {
		return nil, fmt.Errorf("pato: %w", err)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)
	return rgba, nil
}

func readVec(bin []byte, doc gltfDoc, accessor, n int) ([]vec3, error) {
	acc, stride, raw, err := view(bin, doc, accessor)
	if err != nil {
		return nil, err
	}
	if acc.ComponentType != 5126 || (n == 3 && acc.Type != "VEC3") || (n == 2 && acc.Type != "VEC2") {
		return nil, fmt.Errorf("pato: accessor %d is not float vec", accessor)
	}
	if stride == 0 {
		stride = n * 4
	}
	out := make([]vec3, acc.Count)
	for i := range out {
		at := i * stride
		if at+n*4 > len(raw) {
			return nil, fmt.Errorf("pato: accessor %d short", accessor)
		}
		out[i].x = f32(raw[at : at+4])
		out[i].y = f32(raw[at+4 : at+8])
		if n == 3 {
			out[i].z = f32(raw[at+8 : at+12])
		}
	}
	return out, nil
}

func readIndex(bin []byte, doc gltfDoc, accessor int) ([]int, error) {
	acc, stride, raw, err := view(bin, doc, accessor)
	if err != nil {
		return nil, err
	}
	if acc.Type != "SCALAR" {
		return nil, fmt.Errorf("pato: indices are not scalar")
	}
	width := 0
	switch acc.ComponentType {
	case 5123:
		width = 2
	case 5125:
		width = 4
	default:
		return nil, fmt.Errorf("pato: index type %d", acc.ComponentType)
	}
	if stride == 0 {
		stride = width
	}
	out := make([]int, acc.Count)
	for i := range out {
		at := i * stride
		if at+width > len(raw) {
			return nil, fmt.Errorf("pato: indices short")
		}
		if width == 2 {
			out[i] = int(binary.LittleEndian.Uint16(raw[at : at+2]))
		} else {
			out[i] = int(binary.LittleEndian.Uint32(raw[at : at+4]))
		}
	}
	return out, nil
}

func view(bin []byte, doc gltfDoc, accessor int) (gltfAccessor, int, []byte, error) {
	if accessor < 0 || accessor >= len(doc.Accessors) {
		return gltfAccessor{}, 0, nil, fmt.Errorf("pato: accessor %d", accessor)
	}
	acc := doc.Accessors[accessor]
	if acc.BufferView < 0 || acc.BufferView >= len(doc.BufferViews) || acc.Count < 1 {
		return gltfAccessor{}, 0, nil, fmt.Errorf("pato: accessor %d view", accessor)
	}
	bv := doc.BufferViews[acc.BufferView]
	start := bv.ByteOffset + acc.ByteOffset
	if start < 0 || bv.ByteLength < 0 || bv.ByteOffset+bv.ByteLength > len(bin) || start > len(bin) {
		return gltfAccessor{}, 0, nil, fmt.Errorf("pato: accessor %d range", accessor)
	}
	return acc, bv.ByteStride, bin[start:], nil
}

// stage stands the scan in front of the card camera:
// about 1.05 tall, feet just above the contact shadow, centered on x and z.
func stage(m *mesh) {
	if len(m.v) == 0 {
		return
	}
	min := m.v[0].p
	max := min
	for _, v := range m.v[1:] {
		min = vmin(min, v.p)
		max = vmax(max, v.p)
	}
	height := max.y - min.y
	if height < 1e-5 {
		return
	}
	const want = 1.05
	scale := want / height
	center := vec3{(min.x + max.x) / 2, (min.y + max.y) / 2, (min.z + max.z) / 2}
	minY := float32(math.MaxFloat32)
	for i := range m.v {
		p := add(mul(sub(m.v[i].p, center), scale), center)
		m.v[i].p = p
		if p.y < minY {
			minY = p.y
		}
	}
	drop := float32(-0.06) - minY
	var sum vec3
	for i := range m.v {
		m.v[i].p.x -= center.x
		m.v[i].p.y += drop
		m.v[i].p.z -= center.z
		sum = add(sum, m.v[i].p)
	}
	m.pivot = mul(sum, 1/float32(len(m.v)))
	// Gif frame 0 is the side with the beak to the right, and the turn
	// goes toward the tail before it comes around to the front.
	dir := norm(vec3{1.5, 0.26, 1.15})
	m.aim = m.pivot
	m.eye = add(m.pivot, mul(dir, 1.95))
	// Frame 0 of the gif has the head on the right. The turn then carries
	// it toward the tail.
	m.yaw0 = float32(math.Pi/2 - 0.42)
	m.yawSign = -1
}

// dropFins removes the scan's flipper sheet. Those triangles are a flat
// remnant on the ground, and addBlades replaces them with the gif pose.
func dropFins(m *mesh) {
	if m.tex == nil || len(m.i) == 0 {
		return
	}
	kept := m.i[:0]
	for _, tri := range m.i {
		n := 0
		for k := 0; k < 3; k++ {
			v := m.v[tri[k]]
			if isFin(m.tex, v.u, v.v) {
				n++
			}
		}
		if n >= 2 {
			continue
		}
		kept = append(kept, tri)
	}
	m.i = kept
	compact(m)
}

// addBlades hangs two green swim fins under the body, splayed the way the gif holds them.
func addBlades(m *mesh) {
	if len(m.v) == 0 {
		return
	}
	n := len(m.v)
	var sum vec3
	maxY := float32(-1e9)
	minY := float32(math.MaxFloat32)
	for i := 0; i < n; i++ {
		p := m.v[i].p
		sum = add(sum, p)
		if p.y > maxY {
			maxY = p.y
		}
		if p.y < minY {
			minY = p.y
		}
	}
	body := mul(sum, 1/float32(n))
	var headSum vec3
	var headN int
	for i := 0; i < n; i++ {
		p := m.v[i].p
		if p.y < maxY-0.28 {
			continue
		}
		headSum = add(headSum, p)
		headN++
	}
	if headN == 0 {
		return
	}
	head := mul(headSum, 1/float32(headN))
	fwd := norm(vec3{head.x - body.x, 0, head.z - body.z})
	if fwd == (vec3{}) {
		return
	}
	lat := vec3{fwd.z, 0, -fwd.x}
	up := vec3{0, 1, 0}
	foot := add(body, mul(fwd, 0.02))
	foot.y = minY + 0.03
	green := rgb{36, 214, 28}
	boot := rgb{16, 16, 16}
	place := func(side float32) func(vec3) vec3 {
		return func(p vec3) vec3 {
			c, s := float32(math.Cos(float64(side*0.40))), float32(math.Sin(float64(side*0.40)))
			x := c*p.x + s*p.z
			z := -s*p.x + c*p.z
			y := p.y
			cp, sp := float32(math.Cos(-0.32)), float32(math.Sin(-0.32))
			y2 := cp*y - sp*z
			z2 := sp*y + cp*z
			return add(foot, add(add(mul(lat, x+side*0.13), mul(up, y2)), mul(fwd, z2)))
		}
	}
	for _, side := range []float32{-1, 1} {
		xf := place(side)
		base := len(m.v)
		m.box(vec3{0, 0, 0.46}, vec3{0.12, 0.018, 0.50}, xf, green)
		m.box(vec3{0, 0.012, 0.04}, vec3{0.10, 0.032, 0.09}, xf, boot)
		for i := base; i < len(m.v); i++ {
			m.v[i].u = -1
		}
	}
}

func isFin(tex *image.RGBA, u, v float32) bool {
	r, g, b := atlas(tex, u, v)
	return b > 90 && int(b) > int(r)+25 && int(g)+15 > int(r)
}

func atlas(tex *image.RGBA, u, v float32) (byte, byte, byte) {
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
	return p[o], p[o+1], p[o+2]
}

func vmin(a, b vec3) vec3 {
	return vec3{minf(a.x, b.x), minf(a.y, b.y), minf(a.z, b.z)}
}

func vmax(a, b vec3) vec3 {
	return vec3{maxf(a.x, b.x), maxf(a.y, b.y), maxf(a.z, b.z)}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func u32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }

func f32(b []byte) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(b)) }
