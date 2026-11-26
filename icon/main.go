package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"

	"golang.org/x/image/vector"
)

const size = 1024

type pt struct{ x, y float64 }

func main() {
	out := flag.String("o", "icon.png", "output PNG")
	svgOut := flag.String("svg", "icon.svg", "output SVG, empty to skip")
	flag.Parse()

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	body := roundRect(100, 100, 824, 824, 185)
	fill(img, body, vgrad{top: rgb(0x2e, 0x33, 0x3d), bottom: rgb(0x16, 0x19, 0x1f), y0: 100, y1: 924})
	fill(img, roundRect(100, 100, 824, 824, 185), glow{cx: 512, cy: 500, r: 380, c: rgb(0x51, 0xaf, 0xef), a: 0.10})

	const R, w, gap = 250.0, 46.0, 24.0
	r := R - w - gap
	open := 42 * math.Pi / 180
	cx, cy := 512.0+36, 512.0

	glyph := hgrad{left: rgb(0xc6, 0x78, 0xdd), right: rgb(0x51, 0xaf, 0xef), x0: cx - R, x1: cx + R}

	var shapes [][]pt
	shapes = append(shapes, arc(cx, cy, R, w, -open, -2*math.Pi+open))
	px, ph := cx-r+30, 82.0
	shapes = append(shapes, stroke(pt{px, cy - ph}, pt{px + ph, cy}, w))
	shapes = append(shapes, stroke(pt{px + ph, cy}, pt{px, cy + ph}, w))
	shapes = append(shapes, rect(px+ph+40, cy+ph-w/2, 110, w))
	for _, s := range shapes {
		fill(img, s, glyph)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	if *svgOut != "" {
		if err := os.WriteFile(*svgOut, []byte(svg(cx, cy, R, w, open, px, ph)), 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

func svg(cx, cy, R, w, open, px, ph float64) string {
	arcPath := func(rad float64) string {
		x, y := cx+rad*math.Cos(open), cy-rad*math.Sin(open)
		return fmt.Sprintf(`<path d="M %.1f %.1f A %.1f %.1f 0 1 0 %.1f %.1f"/>`, x, y, rad, rad, x, cy+rad*math.Sin(open))
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <defs>
    <linearGradient id="bg" x1="0" y1="100" x2="0" y2="924" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#2e333d"/>
      <stop offset="1" stop-color="#16191f"/>
    </linearGradient>
    <radialGradient id="glow" cx="512" cy="500" r="380" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#51afef" stop-opacity="0.10"/>
      <stop offset="1" stop-color="#51afef" stop-opacity="0"/>
    </radialGradient>
    <linearGradient id="ink" x1="%.1f" y1="0" x2="%.1f" y2="0" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#c678dd"/>
      <stop offset="1" stop-color="#51afef"/>
    </linearGradient>
  </defs>
  <rect x="100" y="100" width="824" height="824" rx="185" fill="url(#bg)"/>
  <rect x="100" y="100" width="824" height="824" rx="185" fill="url(#glow)"/>
  <g fill="none" stroke="url(#ink)" stroke-width="%.1f">
    %s
    <path d="M %.1f %.1f L %.1f %.1f L %.1f %.1f" stroke-linecap="square" stroke-linejoin="miter"/>
  </g>
  <rect x="%.1f" y="%.1f" width="110" height="%.1f" fill="url(#ink)"/>
</svg>
`, cx-R, cx+R, w, arcPath(R), px, cy-ph, px+ph, cy, px, cy+ph, px+ph+40, cy+ph-w/2, w)
}

func arc(cx, cy, r, w, from, to float64) []pt {
	const steps = 256
	ro, ri := r+w/2, r-w/2
	var p []pt
	for i := 0; i <= steps; i++ {
		a := from + (to-from)*float64(i)/steps
		p = append(p, pt{cx + ro*math.Cos(a), cy + ro*math.Sin(a)})
	}
	for i := steps; i >= 0; i-- {
		a := from + (to-from)*float64(i)/steps
		p = append(p, pt{cx + ri*math.Cos(a), cy + ri*math.Sin(a)})
	}
	return p
}

func stroke(a, b pt, w float64) []pt {
	dx, dy := b.x-a.x, b.y-a.y
	l := math.Hypot(dx, dy)
	nx, ny := -dy/l*w/2, dx/l*w/2
	ex, ey := dx/l*w/2, dy/l*w/2
	return []pt{
		{a.x + nx - ex, a.y + ny - ey}, {b.x + nx + ex, b.y + ny + ey},
		{b.x - nx + ex, b.y - ny + ey}, {a.x - nx - ex, a.y - ny - ey},
	}
}

func rect(x, y, w, h float64) []pt {
	return []pt{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}}
}

func roundRect(x, y, w, h, r float64) []pt {
	var p []pt
	corner := func(cx, cy, start float64) {
		for i := 0; i <= 32; i++ {
			a := start + math.Pi/2*float64(i)/32
			p = append(p, pt{cx + r*math.Cos(a), cy + r*math.Sin(a)})
		}
	}
	corner(x+w-r, y+r, -math.Pi/2)
	corner(x+w-r, y+h-r, 0)
	corner(x+r, y+h-r, math.Pi/2)
	corner(x+r, y+r, math.Pi)
	return p
}

func fill(dst *image.RGBA, poly []pt, src image.Image) {
	z := vector.NewRasterizer(size, size)
	z.MoveTo(float32(poly[0].x), float32(poly[0].y))
	for _, q := range poly[1:] {
		z.LineTo(float32(q.x), float32(q.y))
	}
	z.ClosePath()
	z.DrawOp = 0
	z.Draw(dst, dst.Bounds(), src, image.Point{})
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 0xff} }

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 0xff}
}

type vgrad struct {
	top, bottom color.RGBA
	y0, y1      float64
}

func (g vgrad) ColorModel() color.Model { return color.RGBAModel }
func (g vgrad) Bounds() image.Rectangle { return image.Rect(-1e9, -1e9, 1e9, 1e9) }
func (g vgrad) At(x, y int) color.Color {
	return mix(g.top, g.bottom, (float64(y)-g.y0)/(g.y1-g.y0))
}

type hgrad struct {
	left, right color.RGBA
	x0, x1      float64
}

func (g hgrad) ColorModel() color.Model { return color.RGBAModel }
func (g hgrad) Bounds() image.Rectangle { return image.Rect(-1e9, -1e9, 1e9, 1e9) }
func (g hgrad) At(x, y int) color.Color {
	return mix(g.left, g.right, (float64(x)-g.x0)/(g.x1-g.x0))
}

type glow struct {
	cx, cy, r float64
	c         color.RGBA
	a         float64
}

func (g glow) ColorModel() color.Model { return color.RGBAModel }
func (g glow) Bounds() image.Rectangle { return image.Rect(-1e9, -1e9, 1e9, 1e9) }
func (g glow) At(x, y int) color.Color {
	d := math.Hypot(float64(x)-g.cx, float64(y)-g.cy) / g.r
	k := g.a * math.Max(0, 1-d*d)
	return color.RGBA{uint8(float64(g.c.R) * k), uint8(float64(g.c.G) * k), uint8(float64(g.c.B) * k), uint8(255 * k)}
}
