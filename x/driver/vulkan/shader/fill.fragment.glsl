#version 450
layout(set = 0, binding = 0) readonly buffer Inst { vec4 data[]; };
layout(push_constant) uniform Push { vec2 extent; int swapRB; } push;
layout(location = 0) flat in int instance;
layout(location = 0) out vec4 outColor;
void main() {
    int base = instance * 4;
    vec4 box = data[base];
    vec4 color = data[base + 1];
    vec4 rad = data[base + 2];
    float clipH = data[base + 3].x;
    vec2 pixel = gl_FragCoord.xy;
    float halfv = 0.5;
    float localX = pixel.x - box.x + halfv - box.z * halfv;
    float localY = pixel.y - box.y + halfv - box.w * halfv;
    float halfW = box.z * halfv;
    float halfH = box.w * halfv;
    float radius = rad.x;
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
    if (rad.w >= halfv) {
        bool insideX = pixel.x >= rad.y && pixel.x < rad.y + rad.w;
        bool insideY = pixel.y >= rad.z && pixel.y < rad.z + clipH;
        if (!insideX || !insideY) cover = 0.0;
    }
    float a = cover * color.a * (1.0 / 255.0);
    vec3 rgb = color.rgb * (1.0 / 255.0) * a;
    if (push.swapRB != 0) rgb = rgb.bgr;
    outColor = vec4(rgb, a);
}
