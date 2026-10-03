#version 450
layout(set = 0, binding = 0) readonly buffer Pix { uint words[]; };
layout(push_constant) uniform Push { ivec2 size; int swapRB; } push;
layout(location = 0) out vec4 outColor;
void main() {
    ivec2 c = ivec2(gl_FragCoord.xy);
    if (c.x < 0 || c.y < 0 || c.x >= push.size.x || c.y >= push.size.y) {
        outColor = vec4(0.0);
        return;
    }
    uint w = words[c.y * push.size.x + c.x];
    float r = float(w & 255u) / 255.0;
    float g = float((w >> 8u) & 255u) / 255.0;
    float b = float((w >> 16u) & 255u) / 255.0;
    float a = float((w >> 24u) & 255u) / 255.0;
    if (r == 0.0 && g == 0.0 && b == 0.0 && a == 0.0) {
        outColor = vec4(0.0);
        return;
    }
    if (a == 0.0) a = 1.0;
    vec3 rgb = vec3(r, g, b) * a;
    if (push.swapRB != 0) rgb = rgb.bgr;
    outColor = vec4(rgb, a);
}
