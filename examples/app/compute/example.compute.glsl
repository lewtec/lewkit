#version 450
layout(local_size_x = 8, local_size_y = 8) in;
layout(set = 0, binding = 0) buffer Pixels { uint px[]; };
layout(set = 0, binding = 1) buffer Params { uint w, h; float t; uint frame; };

void main() {
    uvec2 p = gl_GlobalInvocationID.xy;
    if (p.x >= w || p.y >= h) return;
    vec2 uv = vec2(p) / vec2(w, h);
    vec3 col = vec3(uv, 0.5 + 0.5 * sin(t));
    uvec3 c = uvec3(clamp(col, 0.0, 1.0) * 255.0);
    px[p.y * w + p.x] = c.r | (c.g << 8) | (c.b << 16) | 0xFF000000;
}
