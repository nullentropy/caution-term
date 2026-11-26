package term

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	caution "github.com/nullentropy/caution"
)

const selectionBg = "#3e4f6a"

type cellPos struct{ row, col int }

func (a cellPos) before(b cellPos) bool {
	return a.row < b.row || a.row == b.row && a.col < b.col
}

type selection struct {
	on, dragging bool
	anchor, head cellPos
	unit         int
}

func (s selection) bounds() (cellPos, cellPos) {
	if s.head.before(s.anchor) {
		return s.head, s.anchor
	}
	return s.anchor, s.head
}

func (s selection) contains(row, col int) bool {
	if !s.on {
		return false
	}
	a, b := s.bounds()
	p := cellPos{row, col}
	return !p.before(a) && !b.before(p)
}

func (t *Term) scroll(lines int) {
	t.mu.Lock()
	sb := 0
	if t.emu != nil && !t.emu.IsAltScreen() {
		sb = t.emu.ScrollbackLen()
	}
	t.mu.Unlock()
	off := min(max(t.offset+lines, 0), sb)
	if off != t.offset {
		t.offset = off
		t.render()
	}
}

func (t *Term) scrollKey(key string) bool {
	t.mu.Lock()
	alt := t.emu == nil || t.emu.IsAltScreen()
	page := max(t.rows-1, 1)
	t.mu.Unlock()
	if alt {
		return false
	}
	switch key {
	case "PageUp":
		t.scroll(page)
	case "PageDown":
		t.scroll(-page)
	case "Home":
		t.scroll(1 << 30)
	case "End":
		t.scroll(-(1 << 30))
	default:
		return false
	}
	return true
}

func (t *Term) selectMouse(m caution.TermMouse) {
	p := cellPos{t.top + m.Y, m.X}
	switch m.Kind {
	case "press":
		t.sel = selection{anchor: p, head: p, dragging: true, unit: max(m.Count, 1)}
		if t.sel.unit > 1 {
			t.sel.on = true
			t.extend()
		}
	case "drag":
		if !t.sel.dragging {
			return
		}
		t.sel.head = p
		t.sel.on = t.sel.unit > 1 || t.sel.anchor != p
		t.extend()
	case "release":
		t.sel.dragging = false
	default:
		return
	}
	t.render()
}

func (t *Term) extend() {
	if t.sel.unit < 2 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.emu == nil {
		return
	}
	a, b := t.sel.bounds()
	if t.sel.unit == 2 {
		a.col, _ = t.wordAt(a)
		_, b.col = t.wordAt(b)
	} else {
		a.col, b.col = 0, t.emu.Width()-1
	}
	if t.sel.head.before(t.sel.anchor) {
		t.sel.head, t.sel.anchor = a, b
	} else {
		t.sel.anchor, t.sel.head = a, b
	}
}

func isWordCell(s string) bool {
	return s != "" && s != " " && !strings.ContainsAny(s, "\"'`()[]{}<>|,;")
}

func (t *Term) wordAt(p cellPos) (int, int) {
	w := t.emu.Width()
	text := func(x int) string {
		if c := t.cellAt(p.row, x); c != nil {
			return c.Content
		}
		return ""
	}
	if !isWordCell(text(p.col)) {
		return p.col, p.col
	}
	l, r := p.col, p.col
	for l > 0 && isWordCell(text(l-1)) {
		l--
	}
	for r < w-1 && isWordCell(text(r+1)) {
		r++
	}
	return l, r
}

func (t *Term) cellAt(row, col int) *uv.Cell {
	sb := t.scrollbackLen()
	if row < sb {
		return t.emu.ScrollbackCellAt(col, row)
	}
	return t.emu.CellAt(col, row-sb)
}

func (t *Term) scrollbackLen() int {
	if t.emu.IsAltScreen() {
		return 0
	}
	return t.emu.ScrollbackLen()
}

func (t *Term) HasSelection() bool { return t.sel.on }

func (t *Term) SelectedText() string {
	if !t.sel.on {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.emu == nil {
		return ""
	}
	a, b := t.sel.bounds()
	w := t.emu.Width()
	var lines []string
	for row := a.row; row <= b.row; row++ {
		from, to := 0, w-1
		if row == a.row {
			from = a.col
		}
		if row == b.row {
			to = b.col
		}
		var sb strings.Builder
		for x := from; x <= to && x < w; x++ {
			c := t.cellAt(row, x)
			if c == nil {
				break
			}
			if c.Content == "" {
				if c.Width == 0 && x > 0 {
					continue
				}
				sb.WriteByte(' ')
				continue
			}
			sb.WriteString(c.Content)
		}
		lines = append(lines, strings.TrimRight(sb.String(), " "))
	}
	return strings.Join(lines, "\n")
}

func (t *Term) CopySelection() bool {
	text := t.SelectedText()
	if text == "" {
		return false
	}
	t.s.WriteClipboard(text)
	return true
}

func (t *Term) ClearScrollback() {
	t.mu.Lock()
	if t.emu != nil {
		t.emu.ClearScrollback()
	}
	t.mu.Unlock()
	t.offset, t.lastSb, t.sel = 0, 0, selection{}
	t.render()
}
