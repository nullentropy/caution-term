package term

import caution "github.com/nullentropy/caution"

type Look struct {
	Name string
	FX   *caution.FX
}

const hashGLSL = `
float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
float luma(vec3 c) { return dot(c, vec3(0.299, 0.587, 0.114)); }
`

var Looks = []Look{
	{Name: "None"},
	{Name: "CRT", FX: &caution.FX{Animate: true, Frag: hashGLSL + `
vec4 effect(vec2 uv) {
  vec2 c = uv * 2.0 - 1.0;
  c *= 1.0 + 0.045 * dot(c, c);
  vec2 s = (c + 1.0) * 0.5;
  if (s.x < 0.0 || s.x > 1.0 || s.y < 0.0 || s.y > 1.0) return vec4(0.0, 0.0, 0.0, 1.0);
  float d = s.y - fract(u_time * 0.07);
  float band = exp(-d * d * 450.0);
  s.x += band * 0.003 * sin(u_time * 3.0 + s.y * 40.0);
  float ab = 0.0018 * (0.25 + dot(c, c));
  vec4 col = src(s);
  col.r = src(s + vec2(ab, 0.0)).r;
  col.b = src(s - vec2(ab, 0.0)).b;
  vec2 o = vec2(1.5) / u_res;
  vec3 glow = (src(s + vec2(o.x, 0.0)).rgb + src(s - vec2(o.x, 0.0)).rgb
             + src(s + vec2(0.0, o.y)).rgb + src(s - vec2(0.0, o.y)).rgb) / 4.0;
  col.rgb += glow * 0.35;
  col.rgb *= 0.82 + 0.18 * sin(s.y * u_res.y * 3.14159);
  col.rgb *= 1.0 + band * 0.25;
  col.rgb *= 0.97 + 0.02 * hash(vec2(floor(u_time * 24.0), s.y));
  col.rgb *= 1.0 - 0.28 * dot(c, c);
  return vec4(col.rgb, 1.0);
}`}},
	{Name: "Green Phosphor", FX: phosphor("vec3(0.30, 1.0, 0.45)")},
	{Name: "Amber Phosphor", FX: phosphor("vec3(1.0, 0.68, 0.18)")},
	{Name: "Bloom", FX: &caution.FX{Frag: hashGLSL + `
vec4 effect(vec2 uv) {
  vec4 base = src(uv);
  vec3 acc = vec3(0.0);
  float total = 0.0;
  for (int i = -3; i <= 3; i++) {
    for (int j = -3; j <= 3; j++) {
      vec2 o = vec2(float(i), float(j)) * 2.2 / u_res;
      vec3 c = src(uv + o).rgb;
      float w = exp(-float(i * i + j * j) / 6.0);
      acc += c * smoothstep(0.45, 0.9, luma(c)) * w;
      total += w;
    }
  }
  return vec4(base.rgb + acc / total * 1.6, 1.0);
}`}},
	{Name: "Chromatic", FX: &caution.FX{Animate: true, Frag: `
vec4 effect(vec2 uv) {
  vec2 c = uv - 0.5;
  float amt = 0.0025 + 0.0015 * sin(u_time * 0.8);
  vec2 dir = c * amt * 4.0;
  float r = src(uv + dir).r;
  float g = src(uv).g;
  float b = src(uv - dir).b;
  return vec4(r, g, b, 1.0);
}`}},
	{Name: "Vaporwave", FX: &caution.FX{Animate: true, Frag: hashGLSL + `
vec4 effect(vec2 uv) {
  vec3 col = src(uv).rgb;
  float l = luma(col);
  vec3 pink = vec3(1.0, 0.35, 0.75);
  vec3 cyan = vec3(0.25, 0.95, 1.0);
  vec3 sky = mix(vec3(0.16, 0.05, 0.25), vec3(0.03, 0.06, 0.18), uv.y);
  float t = 0.5 + 0.5 * sin(u_time * 0.4 + uv.x * 3.0 + uv.y * 2.0);
  vec3 ink = mix(pink, cyan, t) * smoothstep(0.15, 0.7, l);
  vec3 outc = sky + ink * 1.3;
  float grid = step(0.97, fract(uv.y * 40.0 + u_time * 0.5)) * smoothstep(0.6, 1.0, uv.y);
  outc += pink * grid * 0.15;
  outc *= 0.92 + 0.08 * sin(uv.y * u_res.y * 3.14159);
  return vec4(outc, 1.0);
}`}},
	{Name: "Underwater", FX: &caution.FX{Animate: true, Frag: `
vec4 effect(vec2 uv) {
  vec2 w = vec2(sin(uv.y * 18.0 + u_time * 1.6), cos(uv.x * 14.0 + u_time * 1.3)) * 0.0022;
  vec3 col = src(uv + w).rgb;
  float caust = pow(0.5 + 0.5 * sin(uv.x * 30.0 + sin(uv.y * 22.0 + u_time) * 2.0 + u_time * 1.5), 8.0);
  col = mix(col, col * vec3(0.55, 0.9, 1.05), 0.6);
  col += vec3(0.05, 0.12, 0.14) * caust;
  return vec4(col, 1.0);
}`}},
	{Name: "Matrix Rain", FX: &caution.FX{Animate: true, Frag: hashGLSL + `
vec4 effect(vec2 uv) {
  vec3 col = src(uv).rgb;
  float cols = u_res.x / 9.0;
  float cx = floor(uv.x * cols);
  float speed = 0.15 + 0.35 * hash(vec2(cx, 1.0));
  float head = fract(u_time * speed + hash(vec2(cx, 7.0)));
  float d = head - uv.y;
  float tail = d > 0.0 ? exp(-d * 9.0) : 0.0;
  float cy = floor(uv.y * u_res.y / 14.0);
  float glyph = step(0.45, hash(vec2(cx, cy + floor(u_time * 8.0 * speed))));
  float lit = tail * glyph * smoothstep(0.35, 0.0, luma(col));
  col += vec3(0.1, 0.9, 0.35) * lit * 0.35;
  return vec4(col, 1.0);
}`}},
	{Name: "Thermal", FX: &caution.FX{Frag: hashGLSL + `
vec3 heat(float t) {
  return clamp(vec3(1.5 * t - 0.2, 2.0 * t - 0.9, 0.8 - 2.5 * abs(t - 0.25)), 0.0, 1.0);
}
vec4 effect(vec2 uv) {
  float l = luma(src(uv).rgb);
  return vec4(heat(smoothstep(0.1, 0.85, l)), 1.0);
}`}},
	{Name: "Glitch", FX: &caution.FX{Animate: true, Frag: hashGLSL + `
vec4 effect(vec2 uv) {
  float t = floor(u_time * 12.0);
  float row = floor(uv.y * 48.0);
  float burst = step(0.985, hash(vec2(t, 3.0)));
  float shift = (hash(vec2(row, t)) - 0.5) * 0.04 * burst * step(0.6, hash(vec2(row, t + 1.0)));
  vec2 s = vec2(uv.x + shift, uv.y);
  vec3 col = vec3(src(s + vec2(0.003 * burst, 0.0)).r, src(s).g, src(s - vec2(0.003 * burst, 0.0)).b);
  col += (hash(uv * u_res + t) - 0.5) * 0.03;
  return vec4(col, 1.0);
}`}},
}

func phosphor(tint string) *caution.FX {
	return &caution.FX{Animate: true, Frag: hashGLSL + `
vec4 effect(vec2 uv) {
  vec3 tint = ` + tint + `;
  float l = luma(src(uv).rgb);
  vec2 o = vec2(1.5) / u_res;
  float glow = (luma(src(uv + vec2(o.x, 0.0)).rgb) + luma(src(uv - vec2(o.x, 0.0)).rgb)
              + luma(src(uv + vec2(0.0, o.y)).rgb) + luma(src(uv - vec2(0.0, o.y)).rgb)) / 4.0;
  float v = smoothstep(0.12, 0.85, l) + glow * 0.35;
  vec3 col = tint * v + tint * 0.025;
  col *= 0.8 + 0.2 * sin(uv.y * u_res.y * 3.14159);
  col *= 0.97 + 0.03 * hash(vec2(floor(u_time * 30.0), uv.y));
  vec2 c = uv * 2.0 - 1.0;
  col *= 1.0 - 0.25 * dot(c, c);
  return vec4(col, 1.0);
}`}
}

func LookIndex(name string) int {
	for i, s := range Looks {
		if s.Name == name {
			return i
		}
	}
	return 0
}
