package metal

// ShaderSource is the Metal shading language for one GUI frame.
// The fill coverage matches the Vulkan fill shader and present.Composite:
// pixel centers, a hard rounded rect, then an optional clip rectangle.
// swapRB is 1 when the drawable is BGRA8.
const ShaderSource = `
#include <metal_stdlib>
using namespace metal;

struct Push {
    float extentX;
    float extentY;
    int swapRB;
    int pad;
};

struct InkPush {
    int sizeX;
    int sizeY;
    int swapRB;
    int pad;
};

struct FillOut {
    float4 position [[position]];
    uint instance [[flat]];
};

vertex float4 clear_vert(uint vid [[vertex_id]]) {
    float2 unit = float2(float(vid & 1u), float(vid >> 1u));
    return float4(unit.x * 2.0 - 1.0, 1.0 - unit.y * 2.0, 0.0, 1.0);
}

fragment float4 clear_frag() {
    return float4(0.0, 0.0, 0.0, 1.0);
}

vertex FillOut fill_vert(uint vid [[vertex_id]], uint iid [[instance_id]],
                         constant float4* data [[buffer(0)]],
                         constant Push& push [[buffer(1)]]) {
    float4 box = data[iid * 4u];
    float2 unit = float2(float(vid & 1u), float(vid >> 1u));
    float2 pos = box.xy + unit * box.zw;
    float2 ndc = float2(pos.x / push.extentX * 2.0 - 1.0, 1.0 - pos.y / push.extentY * 2.0);
    FillOut out;
    out.position = float4(ndc, 0.0, 1.0);
    out.instance = iid;
    return out;
}

fragment float4 fill_frag(FillOut in [[stage_in]],
                          constant float4* data [[buffer(0)]],
                          constant Push& push [[buffer(1)]]) {
    uint base = in.instance * 4u;
    float4 box = data[base];
    float4 color = data[base + 1u];
    float4 rad = data[base + 2u];
    float clipH = data[base + 3u].x;
    float2 pixel = in.position.xy;
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
    float3 rgb = color.rgb * (1.0 / 255.0) * a;
    if (push.swapRB != 0) rgb = rgb.bgr;
    return float4(rgb, a);
}

vertex float4 ink_vert(uint vid [[vertex_id]]) {
    float2 unit = float2(float(vid & 1u), float(vid >> 1u));
    return float4(unit.x * 2.0 - 1.0, 1.0 - unit.y * 2.0, 0.0, 1.0);
}

fragment float4 ink_frag(float4 position [[position]],
                         constant uint* words [[buffer(0)]],
                         constant InkPush& push [[buffer(1)]]) {
    int x = int(position.x);
    int y = int(position.y);
    if (x < 0 || y < 0 || x >= push.sizeX || y >= push.sizeY) {
        return float4(0.0);
    }
    uint w = words[y * push.sizeX + x];
    float r = float(w & 255u) / 255.0;
    float g = float((w >> 8u) & 255u) / 255.0;
    float b = float((w >> 16u) & 255u) / 255.0;
    float a = float(w >> 24u) / 255.0;
    if (r == 0.0 && g == 0.0 && b == 0.0 && a == 0.0) {
        return float4(0.0);
    }
    if (a == 0.0) a = 1.0;
    float3 rgb = float3(r, g, b) * a;
    if (push.swapRB != 0) rgb = rgb.bgr;
    return float4(rgb, a);
}
`
