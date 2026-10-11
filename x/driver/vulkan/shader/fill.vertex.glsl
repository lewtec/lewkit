#version 450
layout(set = 0, binding = 0) readonly buffer Inst { vec4 data[]; };
layout(push_constant) uniform Push { vec2 extent; int swapRB; int turn; } push;
layout(location = 0) flat out int instance;
void main() {
    instance = gl_InstanceIndex;
    vec4 box = data[instance * 4];
    vec2 unit = vec2(float(gl_VertexIndex & 1), float(gl_VertexIndex >> 1));
    vec2 pos = box.xy + unit * box.zw;
    vec2 fb = pos;
    float fw = push.extent.x;
    float fh = push.extent.y;
    if (push.turn == 2) {
        fb = vec2(push.extent.y - pos.y, pos.x);
        fw = push.extent.y;
        fh = push.extent.x;
    } else if (push.turn == 8) {
        fb = vec2(pos.y, push.extent.x - pos.x);
        fw = push.extent.y;
        fh = push.extent.x;
    } else if (push.turn == 4) {
        fb = vec2(push.extent.x - pos.x, push.extent.y - pos.y);
    }
    vec2 ndc = vec2(fb.x / fw * 2.0 - 1.0, fb.y / fh * 2.0 - 1.0);
    gl_Position = vec4(ndc, float(push.swapRB) * 0.0, 1.0);
}
