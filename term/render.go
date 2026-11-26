package term

import (
	"fmt"
	"image/color"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	caution "github.com/nullentropy/caution"
)

func (t *Term) render() {
	t.pending.Store(false)
	t.mu.Lock()
	emu := t.emu
	if emu == nil {
		t.mu.Unlock()
		return
	}
	w, h := emu.Width(), emu.Height()
	sb := t.scrollbackLen()
	if t.offset > 0 && sb > t.lastSb {
		t.offset += sb - t.lastSb
	}
	t.lastSb = sb
	t.offset = min(max(t.offset, 0), sb)
	t.top = sb - t.offset
	screen := make([][]caution.TermRun, h)
	area := map[string]int{}
	for y := 0; y < h; y++ {
		screen[y] = t.line(t.top+y, w)
		for _, r := range screen[y] {
			area[r.Bg] += cellCount(r.Text)
		}
	}
	if frame := mostCommon(area); frame != t.frame {
		t.frame = frame
		t.grid.TermColors(t.fg, frame)
		t.w.frameChanged(t)
		clear(t.shown)
	}
	changed := map[int][]caution.TermRun{}
	for y := 0; y < h; y++ {
		runs := t.onFrame(screen[y])
		if !runsEqual(runs, t.shown[y]) {
			changed[y] = runs
			t.shown[y] = runs
		}
	}
	for y := range t.shown {
		if y >= h {
			changed[y] = nil
			delete(t.shown, y)
		}
	}
	pos := emu.CursorPosition()
	style, blink := emu.CursorStyle()
	cur := caution.TermCursor{X: pos.X, Y: pos.Y, Visible: !emu.CursorHidden() && t.offset == 0, Shape: "block", Blink: blink}
	switch style {
	case vt.CursorBar:
		cur.Shape = "bar"
	case vt.CursorUnderline:
		cur.Shape = "underline"
	}
	if c := emu.CellAt(pos.X, pos.Y); c != nil {
		cur.Ch = c.Content
		f, b := t.colors(&c.Style)
		cur.Fg, cur.Bg = f, b
	}
	if cc := emu.CursorColor(); cc != nil {
		cur.Fg = hex(cc)
	}
	t.mu.Unlock()
	t.grid.SetTermLines(changed)
	t.grid.SetTermCursor(cur)
}

func (t *Term) line(row, w int) []caution.TermRun {
	var runs []caution.TermRun
	for x := 0; x < w; x++ {
		c := t.cellAt(row, x)
		if c == nil || c.Width == 0 && c.Content == "" && x > 0 {
			continue
		}
		text := c.Content
		if text == "" {
			text = " "
		}
		f, b := t.colors(&c.Style)
		if t.sel.contains(row, x) {
			b = selectionBg
		}
		r := caution.TermRun{
			X: x, Text: text, Fg: f, Bg: b,
			Bold:   c.Style.Attrs&uv.AttrBold != 0,
			Italic: c.Style.Attrs&uv.AttrItalic != 0,
			Under:  c.Style.Underline != uv.UnderlineNone,
			Strike: c.Style.Attrs&uv.AttrStrikethrough != 0,
		}
		ascii := len(text) == 1 && text[0] < 0x80 && c.Width <= 1
		if n := len(runs); n > 0 && ascii {
			last := &runs[n-1]
			if last.X+len(last.Text) == x && isASCII(last.Text) && sameStyle(*last, r) {
				last.Text += text
				continue
			}
		}
		runs = append(runs, r)
	}
	return runs
}

func (t *Term) onFrame(runs []caution.TermRun) []caution.TermRun {
	out := make([]caution.TermRun, 0, len(runs))
	for _, r := range runs {
		if r.Fg == t.fg {
			r.Fg = ""
		}
		if r.Bg == t.frame {
			r.Bg = ""
			if !r.Under && !r.Strike && strings.TrimLeft(r.Text, " ") == "" {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

var palette = [16]string{
	"#1d1f21", "#cc6666", "#b5bd68", "#f0c674", "#81a2be", "#b294bb", "#8abeb7", "#c5c8c6",
	"#666666", "#d54e53", "#b9ca4a", "#e7c547", "#7aa6da", "#c397d8", "#70c0b1", "#eaeaea",
}

func mostCommon(area map[string]int) string {
	best, n := "", -1
	for k, v := range area {
		if v > n || v == n && k < best {
			best, n = k, v
		}
	}
	return best
}

func cellCount(s string) int { return len([]rune(s)) }

func (t *Term) colors(s *uv.Style) (string, string) {
	f, b := t.resolve(s.Fg), t.resolve(s.Bg)
	if f == "" {
		f = t.fg
	}
	if b == "" {
		b = t.bg
	}
	if s.Attrs&uv.AttrReverse != 0 {
		f, b = b, f
	}
	return f, b
}

func (t *Term) resolve(c color.Color) string {
	switch v := c.(type) {
	case nil:
		return ""
	case ansi.BasicColor:
		return hex(t.emu.IndexedColor(int(v)))
	case ansi.IndexedColor:
		return hex(t.emu.IndexedColor(int(v)))
	}
	return hex(c)
}

func hex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func hexColor(s string) color.Color {
	var r, g, b uint8
	fmt.Sscanf(strings.TrimPrefix(s, "#"), "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 0xff}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func sameStyle(a, b caution.TermRun) bool {
	return a.Fg == b.Fg && a.Bg == b.Bg && a.Bold == b.Bold && a.Italic == b.Italic &&
		a.Under == b.Under && a.Strike == b.Strike
}

func runsEqual(a, b []caution.TermRun) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
