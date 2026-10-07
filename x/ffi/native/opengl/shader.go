//go:build !darwin && !ios

package opengl

// Present shaders are hand-written. The fill coverage matches present.Composite:
// pixel centers, a hard rounded rect, then an optional clip rectangle.
// OpenGL's window origin is the lower left, so the vertex shader flips y and
// the fragment shaders flip gl_FragCoord themselves. Pixel (0, 0) is the top
// row, which is where the CPU picture stores its first row. The flip uses the
// draw size rather than layout(origin_upper_left): Mesa keeps that layout's
// height from the previous framebuffer, so a later larger draw is shifted.
// swapRB stays available for a BGRA drawable. RGBA8 leaves it at 0.

func versionHeader(g *glAPI) string {
	if g != nil && g.gles {
		return "#version 300 es\nprecision highp float;\nprecision highp int;\n"
	}
	// A 4.1 context speaks GLSL 410. 3.3 and newer accept 330.
	if g != nil && g.major == 4 && g.minor < 3 {
		return "#version 410\n"
	}
	return "#version 330 core\n"
}

func fillVertex(g *glAPI) string {
	return versionHeader(g) + `
layout(location = 0) in vec4 aBox;
layout(location = 1) in vec4 aColor;
layout(location = 2) in vec4 aRad;
layout(location = 3) in vec4 aClip;
flat out vec4 vBox;
flat out vec4 vColor;
flat out vec4 vRad;
flat out vec4 vClip;
uniform vec2 uExtent;
void main() {
    vBox = aBox;
    vColor = aColor;
    vRad = aRad;
    vClip = aClip;
    vec2 unit = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
    vec2 pos = aBox.xy + unit * aBox.zw;
    vec2 ndc = vec2(pos.x / uExtent.x * 2.0 - 1.0, 1.0 - pos.y / uExtent.y * 2.0);
    gl_Position = vec4(ndc, 0.0, 1.0);
}
`
}

func fillFragment(g *glAPI) string {
	return versionHeader(g) + `
flat in vec4 vBox;
flat in vec4 vColor;
flat in vec4 vRad;
flat in vec4 vClip;
uniform vec2 uExtent;
uniform int uSwapRB;
out vec4 outColor;
void main() {
    vec2 pixel = vec2(gl_FragCoord.x, uExtent.y - gl_FragCoord.y);
    float halfv = 0.5;
    float localX = pixel.x - vBox.x + halfv - vBox.z * halfv;
    float localY = pixel.y - vBox.y + halfv - vBox.w * halfv;
    float halfW = vBox.z * halfv;
    float halfH = vBox.w * halfv;
    float radius = vRad.x;
    radius = radius < halfW ? radius : halfW;
    radius = radius < halfH ? radius : halfH;
    float cx = abs(localX) - halfW + radius;
    float cy = abs(localY) - halfH + radius;
    float ox = max(cx, 0.0);
    float oy = max(cy, 0.0);
    float cornerMax = max(cx, cy);
    float inside = cornerMax < 0.0 ? cornerMax : 0.0;
    float dist = inside + sqrt(ox * ox + oy * oy) - radius;
    float cover = dist < halfv ? 1.0 : 0.0;
    if (vRad.w >= halfv) {
        bool insideX = pixel.x >= vRad.y && pixel.x < vRad.y + vRad.w;
        bool insideY = pixel.y >= vRad.z && pixel.y < vRad.z + vClip.x;
        if (!insideX || !insideY) cover = 0.0;
    }
    float a = cover * vColor.a * (1.0 / 255.0);
    vec3 rgb = vColor.rgb * (1.0 / 255.0) * a;
    if (uSwapRB != 0) rgb = rgb.bgr;
    outColor = vec4(rgb, a);
}
`
}

func imageVertex(g *glAPI) string {
	return versionHeader(g) + `
void main() {
    vec2 unit = vec2(float(gl_VertexID & 1), float(gl_VertexID >> 1));
    vec2 ndc = vec2(unit.x * 2.0 - 1.0, 1.0 - unit.y * 2.0);
    gl_Position = vec4(ndc, 0.0, 1.0);
}
`
}

func imageFragment(g *glAPI) string {
	return versionHeader(g) + `
uniform sampler2D uPix;
uniform int uSwapRB;
out vec4 outColor;
void main() {
    ivec2 size = textureSize(uPix, 0);
    ivec2 c = ivec2(int(gl_FragCoord.x), size.y - 1 - int(gl_FragCoord.y));
    if (c.x < 0 || c.y < 0 || c.x >= size.x || c.y >= size.y) {
        outColor = vec4(0.0);
        return;
    }
    vec4 t = texelFetch(uPix, c, 0);
    float r = t.r;
    float g = t.g;
    float b = t.b;
    float a = t.a;
    if (r == 0.0 && g == 0.0 && b == 0.0 && a == 0.0) {
        outColor = vec4(0.0);
        return;
    }
    if (a == 0.0) a = 1.0;
    vec3 rgb = vec3(r, g, b) * a;
    if (uSwapRB != 0) rgb = rgb.bgr;
    outColor = vec4(rgb, a);
}
`
}
