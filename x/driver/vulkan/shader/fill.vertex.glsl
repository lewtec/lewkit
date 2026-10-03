#version 450
layout(set = 0, binding = 0) readonly buffer Inst { vec4 data[]; };
layout(push_constant) uniform Push { vec2 extent; int swapRB; } push;
layout(location = 0) flat out int instance;
void main() {
    instance = gl_InstanceIndex;
    vec4 box = data[instance * 4];
    vec2 unit = vec2(float(gl_VertexIndex & 1), float(gl_VertexIndex >> 1));
    vec2 pos = box.xy + unit * box.zw;
    vec2 ndc = vec2(pos.x / push.extent.x * 2.0 - 1.0, pos.y / push.extent.y * 2.0 - 1.0);
    gl_Position = vec4(ndc, float(push.swapRB) * 0.0, 1.0);
}
