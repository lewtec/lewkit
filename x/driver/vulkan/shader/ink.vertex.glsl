#version 450
layout(push_constant) uniform Push { ivec2 size; int swapRB; } push;
void main() {
    vec2 unit = vec2(float(gl_VertexIndex & 1), float(gl_VertexIndex >> 1));
    gl_Position = vec4(unit * 2.0 - 1.0, float(push.swapRB) * 0.0, 1.0);
}
