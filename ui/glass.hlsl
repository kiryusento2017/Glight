// 液态玻璃折射 shader（移植自 shuding/liquid-glass 折射核）。
// 全屏三角顶点 + 折射像素：在像素空间用「圆角矩形 + 连续曲率角」有向距离场
// 限定玻璃形状（苹果味 G2 连续圆角），仅在最外 uRefractBand 像素带做向心收缩折射
// （中心一大片完全不折射，符合苹果液态玻璃），再做轻微色彩微调，叠加三灯。
//
// 画布(CANVAS) > pill(玻璃逻辑尺寸)：画布留 margin 容纳形变（稳态拉伸+过冲），
// 玻璃 pill 居中在画布内，画布空白区 alpha=0 透明。形变由 uScaleX/uScaleY 驱动。
// CANVAS 必须与 ui 包 winW/winH 保持一致。
//
// 视觉参数（圆角/折射/调色/三灯）由 cbuffer 注入，来源 glass-tuning.json 热重载。
// swapchain 为 premultiplied alpha：输出 rgb 需预乘 alpha，玻璃外 alpha=0 透明。

Texture2D    desktop : register(t0);
SamplerState samp    : register(s0);

cbuffer Params : register(b0) {
    float uActive;      // 0灰 1绿 2黄 3红
    float uBlink;       // 0~1 闪烁亮度
    float uScaleX;      // 形变：水平缩放（1=原尺寸）
    float uScaleY;      // 形变：垂直缩放（1=原尺寸）
    float uCornerR;     // 圆角半径（像素）
    float uCornerN;     // 角部连续曲率指数
    float uRefractBand; // 折射带深度（像素）
    float uEdgeSqueeze; // 边缘最强收缩系数
    float uContrast;    // 对比度
    float uBrightness;  // 亮度
    float uSaturate;    // 饱和度
    float uLampR;       // 灯半径（像素）
    float uLampGap;     // 灯间距（像素）
    float uGlow;        // 点亮外发光强度
    float2 _pad;
};

static const float2 CANVAS = float2(240.0, 128.0); // 画布像素尺寸（= winW/winH）：收紧到形变峰值包络，死区最小
static const float2 PILL   = float2(230.0, 96.0);  // 玻璃逻辑尺寸（≈2.4:1）

struct VSOut {
    float4 pos : SV_Position;
    float2 uv  : TEXCOORD0;
};

// 全屏三角：用 SV_VertexID 生成 (0,0),(2,0),(0,2)，铺满整个渲染目标。
VSOut VSMain(uint vid : SV_VertexID) {
    VSOut o;
    float2 uv = float2((vid << 1) & 2, vid & 2); // 0/2 组合
    o.uv = uv;
    o.pos = float4(uv * float2(2.0, -2.0) + float2(-1.0, 1.0), 0.0, 1.0);
    return o;
}

// 圆角矩形有向距离场，四角用 n 范数做连续曲率（n=2 普通圆角，n>2 苹果味 G2 连续）。
// b 为半宽高，r 为圆角半径（须 ≤ min(b)）。返回像素单位 SDF：<0 内部，=0 边界，>0 外部。
float squircleSDF(float2 p, float2 b, float r, float n) {
    float2 q = abs(p) - b + r;
    float2 m = max(q, 0.0);
    float  corner = pow(pow(m.x, n) + pow(m.y, n), 1.0 / n); // n 范数角部距离
    return min(max(q.x, q.y), 0.0) + corner - r;
}

// 叠加一个圆灯：圆盘半不透明覆盖（任何背景都清晰）+ 点亮时外发光 halo。
// 熄灭留暗灯罩、点亮鲜亮灯色；core*0.9 保留一丝折射底色维持玻璃感。
float3 addLamp(float3 base, float2 px, float2 ctr, float r, float3 col, float on, float glowK) {
    float dd   = length(px - ctr);
    float core = smoothstep(r, r - 2.0, dd);   // 圆盘
    float glow = smoothstep(r * 2.2, r, dd);   // 外发光范围
    float3 body = lerp(col * 0.18, col * 1.35, on); // 熄灭暗罩 → 点亮鲜亮
    float3 outc = lerp(base, body, core * 0.9);      // 圆盘内覆盖（不受背景影响）
    outc += col * glow * glowK * on;                 // 点亮时外发光
    return outc;
}

float4 PSMain(VSOut i) : SV_Target {
    float2 px  = i.uv * CANVAS; // 画布像素坐标
    float2 ctr = CANVAS * 0.5;
    float2 p   = px - ctr;      // 相对画布中心

    // 玻璃形状：连续曲率圆角矩形（形变后半尺寸 = pill 半尺寸 × scale）
    float2 half = PILL * 0.5 * float2(uScaleX, uScaleY);
    float  rr   = min(uCornerR, min(half.x, half.y)); // 圆角不得超过半尺寸
    float  d    = squircleSDF(p, half, rr, uCornerN); // <0 内部

    // 折射（移植 shuding/liquid-glass 折射核）：中心一大片清晰平台(scaled=1 零位移)，
    // 仅最外 uRefractBand 深度内向心强收缩(scaled→uEdgeSqueeze)，边缘把外围内容压向中心。
    float  dn     = -d;                                  // 内部深度：边缘 0 → 中心最大
    float  disp   = smoothstep(0.0, uRefractBand, dn);   // 边缘 0 → 内部平台 1
    float  scaled = lerp(uEdgeSqueeze, 1.0, disp);
    float2 suv    = (ctr + p * scaled) / CANVAS;

    float3 c = desktop.Sample(samp, suv).rgb;

    // 轻微色彩微调（brightness / contrast / saturate）
    c *= uBrightness;
    c = (c - 0.5) * uContrast + 0.5;
    float g = dot(c, float3(0.299, 0.587, 0.114));
    c = lerp(float3(g, g, g), c, uSaturate);
    c = saturate(c);

    // 三灯叠加（像素空间正圆，基于画布中心，间距随形变）。绿常亮、红/黄按 uBlink 闪、灰全灭
    float r   = uLampR;
    float gap = uLampGap * uScaleX;
    float onR = (uActive == 3.0) ? uBlink : 0.0;
    float onY = (uActive == 2.0) ? uBlink : 0.0;
    float onG = (uActive == 1.0) ? 1.0    : 0.0;
    c = addLamp(c, px, ctr + float2(-gap, 0.0), r, float3(0.910, 0.188, 0.165), onR, uGlow);
    c = addLamp(c, px, ctr + float2( 0.0, 0.0), r, float3(0.941, 0.753, 0.251), onY, uGlow);
    c = addLamp(c, px, ctr + float2( gap, 0.0), r, float3(0.188, 0.753, 0.251), onG, uGlow);
    c = saturate(c);

    // 软边抗锯齿（fwidth 自适应）：d<0 内部不透明；premultiplied alpha
    float aa   = fwidth(d);
    float mask = 1.0 - smoothstep(-aa, aa, d);
    return float4(c * mask, mask);
}
