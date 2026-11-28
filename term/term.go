package term

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	caution "github.com/nullentropy/caution"
)

type Options struct {
	Argv []string
	Dir  string
	Env  []string
}

type Term struct {
	s    *caution.Session
	w    *Window
	argv []string
	dir  string
	env  []string
	fg   string
	bg   string
	grid *caution.Node

	mu         sync.Mutex
	emu        *vt.Emulator
	ptmx       *os.File
	proc       *os.Process
	appCursor  bool
	mouseModes map[ansi.Mode]bool
	cols, rows int
	restarting bool
	closing    bool

	pending atomic.Bool

	shown  map[int][]caution.TermRun
	title  string
	frame  string
	offset int
	lastSb int
	top    int
	sel    selection
}

func newTerm(w *Window, o Options) *Term {
	t := &Term{s: w.s, w: w, argv: o.Argv, dir: o.Dir, env: o.Env, fg: w.fg, bg: w.bg,
		mouseModes: map[ansi.Mode]bool{}, shown: map[int][]caution.TermRun{}, frame: w.bg}
	t.grid = caution.TermGrid().FontSize(w.size).TermColors(w.fg, w.bg).
		SetProp("pad", 6.0).
		OnGridSize(t.resize).OnTermInput(t.input).OnTermMouse(t.mouse).
		Dock("fill")
	if w.fonts {
		t.grid.Font(family)
	}
	go func() {
		<-t.s.Done()
		t.hangup()
	}()
	return t
}

func (t *Term) Argv() []string { return t.argv }

func (t *Term) Title() string {
	if t.title != "" {
		return t.title
	}
	return filepath.Base(t.argv[0])
}

func (t *Term) hangup() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.proc != nil {
		t.proc.Signal(syscall.SIGHUP)
	}
}

func (t *Term) resize(cols, rows int) {
	t.mu.Lock()
	t.cols, t.rows = cols, rows
	if t.emu != nil {
		t.emu.Resize(cols, rows)
		pty.Setsize(t.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
		t.mu.Unlock()
		t.render()
		return
	}
	t.mu.Unlock()
	t.launch()
}

func (t *Term) launch() {
	if err := t.start(t.cols, t.rows); err != nil {
		t.grid.SetTermLines(map[int][]caution.TermRun{0: {{Text: "could not start " + t.argv[0] + ": " + err.Error()}}})
	}
}

func (t *Term) Restart() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.proc == nil {
		return
	}
	t.restarting = true
	t.proc.Signal(syscall.SIGHUP)
}

func (t *Term) start(cols, rows int) error {
	emu := vt.NewEmulator(cols, rows)
	emu.SetDefaultForegroundColor(hexColor(t.fg))
	emu.SetDefaultBackgroundColor(hexColor(t.bg))
	for i, c := range palette {
		emu.SetIndexedColor(i, hexColor(c))
	}
	emu.SetCallbacks(vt.Callbacks{
		EnableMode:  func(m ansi.Mode) { t.setMode(m, true) },
		DisableMode: func(m ansi.Mode) { t.setMode(m, false) },

		Title: func(title string) {
			go t.s.Update(func() {
				if title != t.title {
					t.title = title
					t.w.titleChanged(t)
				}
			})
		},
	})
	emu.RegisterOscHandler(52, func(data []byte) bool {
		parts := strings.SplitN(string(data), ";", 3)
		if len(parts) == 3 {
			if b, err := base64.StdEncoding.DecodeString(parts[2]); err == nil {
				go t.s.Update(func() { t.s.WriteClipboard(string(b)) })
			}
		}
		return true
	})

	cmd := exec.Command(t.argv[0], t.argv[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	if os.Getenv("LANG") == "" && os.Getenv("LC_ALL") == "" && os.Getenv("LC_CTYPE") == "" {
		cmd.Env = append(cmd.Env, "LANG=en_US.UTF-8")
	}
	cmd.Env = append(cmd.Env, t.env...)
	if t.dir != "" {
		cmd.Dir = t.dir
		cmd.Env = append(cmd.Env, "PWD="+t.dir)
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.emu, t.ptmx, t.proc, t.appCursor = emu, ptmx, cmd.Process, false
	clear(t.mouseModes)
	t.mu.Unlock()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := emu.Read(buf)
			if n > 0 {
				ptmx.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	output := make(chan struct{}, 1)
	go t.paceRenders(output)
	go func() {
		buf := make([]byte, 64<<10)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				t.mu.Lock()
				emu.Write(buf[:n])
				t.mu.Unlock()
				select {
				case output <- struct{}{}:
				default:
				}
			}
			if err != nil {
				break
			}
		}
		close(output)
		cmd.Wait()
		t.mu.Lock()
		again := t.restarting
		t.restarting = false
		t.emu, t.proc = nil, nil
		emu.Close()
		ptmx.Close()
		t.mu.Unlock()
		t.s.Update(func() {
			if again {
				clear(t.shown)
				t.offset, t.lastSb, t.sel = 0, 0, selection{}
				t.launch()
				return
			}
			t.w.exited(t)
		})
	}()
	return nil
}

const (
	renderQuiet = 2 * time.Millisecond
	renderMax   = 16 * time.Millisecond
)

// macos ptys hand over at most 1024 bytes per read so one emacs redraw
// arrives in pieces and rendering after the first shows half a screen
// render once output goes quiet, or every renderMax while it keeps coming
func (t *Term) paceRenders(output <-chan struct{}) {
	quiet := time.NewTimer(time.Hour)
	quiet.Stop()
	for range output {
		deadline := time.After(renderMax)
		quiet.Reset(renderQuiet)
	wait:
		for {
			select {
			case _, ok := <-output:
				if !ok {
					break wait
				}
				quiet.Reset(renderQuiet)
			case <-quiet.C:
				break wait
			case <-deadline:
				quiet.Stop()
				break wait
			}
		}
		if !t.pending.Swap(true) {
			t.s.Update(t.render)
		}
	}
}

func (t *Term) setMode(m ansi.Mode, on bool) {
	switch m {
	case ansi.ModeCursorKeys:
		t.appCursor = on
	case ansi.ModeMouseX10, ansi.ModeMouseNormal, ansi.ModeMouseHighlight,
		ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent:
		t.mouseModes[m] = on
	}
}

func (t *Term) mouseOn() bool {
	for _, on := range t.mouseModes {
		if on {
			return true
		}
	}
	return false
}

func (t *Term) input(in caution.TermInput) {
	if in.Shift && !in.Ctrl && !in.Alt && t.scrollKey(in.Key) {
		return
	}
	if t.offset != 0 || t.sel.on {
		t.offset, t.sel = 0, selection{}
		t.render()
	}
	if in.Paste {
		t.Paste(in.Text)
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.emu == nil {
		return
	}
	if seq := encodeKey(in, t.appCursor); seq != "" {
		t.ptmx.WriteString(seq)
	}
}

func (t *Term) Paste(text string) {
	t.mu.Lock()
	emu := t.emu
	t.mu.Unlock()
	if emu == nil || text == "" {
		return
	}
	go func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		emu.Paste(text)
	}()
}

func (t *Term) mouse(m caution.TermMouse) {
	t.mu.Lock()
	emu := t.emu
	forward := emu != nil && t.mouseOn() && !m.Shift
	alt := emu != nil && emu.IsAltScreen()
	t.mu.Unlock()
	if emu == nil {
		return
	}
	switch {
	case forward:
		if m.Kind == "press" && t.sel.on {
			t.sel = selection{}
			t.render()
		}
		t.sendMouse(emu, m)
	case m.Kind == "wheel" && alt:
		key := "Down"
		n := m.Dy
		if n < 0 {
			key, n = "Up", -n
		}
		for range min(n, 10) {
			t.input(caution.TermInput{Key: key})
		}
	case m.Kind == "wheel":
		t.scroll(-m.Dy)
	case m.Button == 1:
		t.selectMouse(m)
	}
}

func (t *Term) sendMouse(emu *vt.Emulator, m caution.TermMouse) {
	var mod uv.KeyMod
	if m.Ctrl {
		mod |= uv.ModCtrl
	}
	base := uv.Mouse{X: m.X, Y: m.Y, Button: uv.MouseButton(m.Button), Mod: mod}
	var evs []vt.Mouse
	switch m.Kind {
	case "press":
		evs = append(evs, uv.MouseClickEvent(base))
	case "release":
		evs = append(evs, uv.MouseReleaseEvent(base))
	case "drag":
		evs = append(evs, uv.MouseMotionEvent(base))
	case "wheel":
		base.Button = uv.MouseWheelDown
		n := m.Dy
		if n < 0 {
			base.Button, n = uv.MouseWheelUp, -n
		}
		for range min(n, 10) {
			evs = append(evs, uv.MouseWheelEvent(base))
		}
	}
	go func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		for _, ev := range evs {
			emu.SendMouse(ev)
		}
	}()
}
