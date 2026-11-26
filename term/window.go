package term

import (
	caution "github.com/nullentropy/caution"
)

const family = "term"

type WindowOptions struct {
	Fonts    []string
	FontSize float64
	Look     string
	Fg, Bg   string
	OnEmpty  func()
	OnChange func()
}

type Window struct {
	s        *caution.Session
	fg, bg   string
	size     float64
	look     string
	fonts    bool
	onEmpty  func()
	onChange func()

	tabs   []*Term
	active int

	bar   *caution.Node
	strip *caution.Node
	layer *caution.Node
	root  *caution.Node
}

func NewWindow(s *caution.Session, o WindowOptions) *Window {
	if o.Fg == "" {
		o.Fg = "#ffffff"
	}
	if o.Bg == "" {
		o.Bg = "#282c34"
	}
	if o.FontSize == 0 {
		o.FontSize = DefaultFontSize
	}
	w := &Window{s: s, fg: o.Fg, bg: o.Bg, size: ClampFontSize(o.FontSize),
		fonts: len(o.Fonts) > 0, onEmpty: o.OnEmpty, onChange: o.OnChange}
	if w.fonts {
		s.AddFont(family, o.Fonts...)
	}
	w.bar = caution.HStack().Gap(4).Anchor(caution.A{Left: caution.Px(84), CenterY: caution.Px(0)})
	w.strip = caution.Panel().Bg(o.Bg).H(32).Dock("top").WindowDrag().Kids(w.bar)
	w.layer = caution.DockPanel().Kids(w.strip).Dock("fill")
	w.root = caution.DockPanel().Kids(w.layer)
	s.SetTheme(map[string]string{"bg": o.Bg})
	w.SetLook(o.Look)
	return w
}

func (w *Window) Node() *caution.Node { return w.root }

func (w *Window) Open(o Options) *Term {
	t := newTerm(w, o)
	w.tabs = append(w.tabs, t)
	w.Select(len(w.tabs) - 1)
	return t
}

func (w *Window) Active() *Term {
	if len(w.tabs) == 0 {
		return nil
	}
	return w.tabs[w.active]
}

func (w *Window) Tabs() []*Term { return w.tabs }

func (w *Window) Select(i int) {
	if len(w.tabs) == 0 {
		return
	}
	i = ((i % len(w.tabs)) + len(w.tabs)) % len(w.tabs)
	if old := w.Active(); old != nil && old.grid.Parent() != nil && old != w.tabs[i] {
		old.grid.Remove()
	}
	w.active = i
	t := w.tabs[i]
	if t.grid.Parent() == nil {
		w.layer.Add(t.grid)
	}
	t.grid.Focus()
	w.frameChanged(t)
	w.titleChanged(t)
	w.drawTabs()
}

func (w *Window) Step(d int) { w.Select(w.active + d) }

func (w *Window) Close(t *Term) { t.hangup() }

func (w *Window) exited(t *Term) {
	i := -1
	for j, x := range w.tabs {
		if x == t {
			i = j
		}
	}
	if i < 0 {
		return
	}
	t.grid.Remove()
	w.tabs = append(w.tabs[:i], w.tabs[i+1:]...)
	if len(w.tabs) == 0 {
		w.drawTabs()
		if w.onEmpty != nil {
			w.onEmpty()
		}
		return
	}
	if w.active >= i && w.active > 0 {
		w.active--
	}
	w.Select(w.active)
}

func (w *Window) frameChanged(t *Term) {
	if t != w.Active() {
		return
	}
	w.strip.Bg(t.frame)
	w.s.SetTheme(map[string]string{"bg": t.frame})
}

func (w *Window) titleChanged(t *Term) {
	if t == w.Active() {
		w.s.SetTitle(t.Title())
	}
	w.drawTabs()
}

func (w *Window) drawTabs() {
	w.bar.Clear()
	if len(w.tabs) > 1 {
		for i, t := range w.tabs {
			b := caution.Button(tabLabel(t.Title())).OnClick(func() { w.Select(i) })
			if i == w.active {
				b.Primary()
			}
			w.bar.Add(b)
		}
	}
	if w.onChange != nil {
		w.onChange()
	}
}

func tabLabel(s string) string {
	r := []rune(s)
	if len(r) > 24 {
		return string(r[:23]) + "…"
	}
	return s
}

func (w *Window) FontSize() float64 { return w.size }

func (w *Window) SetFontSize(sz float64) {
	w.size = ClampFontSize(sz)
	for _, t := range w.tabs {
		t.grid.FontSize(w.size)
	}
}

func (w *Window) Look() string { return w.look }

func (w *Window) SetLook(name string) {
	l := Looks[LookIndex(name)]
	w.look = l.Name
	if l.FX != nil {
		w.layer.Effect(*l.FX)
	} else {
		w.layer.SetProp("effect", nil)
	}
}
