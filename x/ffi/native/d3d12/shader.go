package d3d12

// ShaderClear, ShaderFill, and ShaderInk are the HLSL for one GUI frame.
// The fill coverage matches the Vulkan fill shader and present.Composite:
// pixel centers, a hard rounded rect, then an optional clip rectangle.
// swapRB stays 0. The swapchain is RGBA8, so shader output is stored as written.
const ShaderClear = `
float4 clear_vert(uint vid : SV_VertexID) : SV_Position {
    float2 unit = float2(float(vid & 1u), float(vid >> 1u));
    return float4(unit.x * 2.0 - 1.0, 1.0 - unit.y * 2.0, 0.0, 1.0);
}

float4 clear_frag() : SV_Target {
    return float4(0.0, 0.0, 0.0, 1.0);
}
`

const ShaderFill = `
cbuffer Push : register(b0) {
    float extentX;
    float extentY;
    int swapRB;
    int pad;
};
StructuredBuffer<float4> data : register(t0);

struct FillOut {
    float4 position : SV_Position;
    nointerpolation uint instance : INSTANCE;
};

FillOut fill_vert(uint vid : SV_VertexID, uint iid : SV_InstanceID) {
    float4 box = data[iid * 4u];
    float2 unit = float2(float(vid & 1u), float(vid >> 1u));
    float2 pos = box.xy + unit * box.zw;
    float2 ndc = float2(pos.x / extentX * 2.0 - 1.0, 1.0 - pos.y / extentY * 2.0);
    FillOut o;
    o.position = float4(ndc, 0.0, 1.0);
    o.instance = iid;
    return o;
}

float4 fill_frag(FillOut pin) : SV_Target {
    uint base = pin.instance * 4u;
    float4 box = data[base];
    float4 color = data[base + 1u];
    float4 rad = data[base + 2u];
    float clipH = data[base + 3u].x;
    float2 pixel = pin.position.xy;
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
    if (swapRB != 0) rgb = rgb.bgr;
    return float4(rgb, a);
}
`

const ShaderInk = `
cbuffer InkPush : register(b0) {
    int sizeX;
    int sizeY;
    int swapRB;
    int pad;
};
StructuredBuffer<uint> words : register(t0);

float4 ink_vert(uint vid : SV_VertexID) : SV_Position {
    float2 unit = float2(float(vid & 1u), float(vid >> 1u));
    return float4(unit.x * 2.0 - 1.0, 1.0 - unit.y * 2.0, 0.0, 1.0);
}

float4 ink_frag(float4 position : SV_Position) : SV_Target {
    int x = int(position.x);
    int y = int(position.y);
    if (x < 0 || y < 0 || x >= sizeX || y >= sizeY) {
        return float4(0.0, 0.0, 0.0, 0.0);
    }
    uint w = words[uint(y) * uint(sizeX) + uint(x)];
    float r = float(w & 255u) / 255.0;
    float g = float((w >> 8u) & 255u) / 255.0;
    float b = float((w >> 16u) & 255u) / 255.0;
    float a = float(w >> 24u) / 255.0;
    if (r == 0.0 && g == 0.0 && b == 0.0 && a == 0.0) {
        return float4(0.0, 0.0, 0.0, 0.0);
    }
    if (a == 0.0) a = 1.0;
    float3 rgb = float3(r, g, b) * a;
    if (swapRB != 0) rgb = rgb.bgr;
    return float4(rgb, a);
}
`
