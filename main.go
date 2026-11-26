package main

import (
	"flag"
	"log"
	"os"
	"os/exec"

	caution "github.com/nullentropy/caution"
	"github.com/nullentropy/caution-term/term"
	"github.com/nullentropy/caution/native"
)

var (
	addr     = flag.String("addr", "", "serve browsers on this address instead of opening a window")
	fontGlob = flag.String("font", term.DefaultFonts, "font files, {a,b} alternatives allowed; empty for the built-in mono")
	look     = flag.String("look", "", "start with this look, by name")
	shot     = flag.String("shot", "", "render offscreen to this PNG after -settle ms and exit")
	settle   = flag.Float64("settle", 3000, "ms to wait before -shot")
)

func main() {
	flag.Parse()
	dir := term.PrepareBundle()
	cfg := term.LoadSettings("Caution Term")
	if *look != "" {
		cfg.Look = *look
	}
	save := func() {
		if *shot == "" {
			cfg.Save()
		}
	}
	argv := flag.Args()
	if len(argv) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/zsh"
		}
		argv = []string{shell, "-l"}
	}
	fonts := term.LoadFonts(*fontGlob)

	mount := func(s *caution.Session) *caution.Node {
		var w *term.Window
		var menu func()
		w = term.NewWindow(s, term.WindowOptions{
			Fonts: fonts, FontSize: cfg.FontSize, Look: cfg.Look,
			OnEmpty: func() {
				if *addr == "" {
					os.Exit(0)
				}
				w.Open(term.Options{Argv: argv, Dir: dir})
			},
			OnChange: func() {
				if menu != nil {
					menu()
				}
			},
		})
		open := func() { w.Open(term.Options{Argv: argv, Dir: dir}) }
		active := func(fn func(t *term.Term)) func() {
			return func() {
				if t := w.Active(); t != nil {
					fn(t)
				}
			}
		}
		menu = func() {
			s.SetMenu(
				caution.Menu{Title: "Shell", Items: []caution.MenuItem{
					{Title: "New Window", Key: "cmd+n", OnPick: newWindow},
					{Title: "New Tab", Key: "cmd+t", OnPick: open},
					{Title: "Close Tab", Key: "cmd+w", OnPick: active(w.Close)},
					{Sep: true},
					{Title: "Next Tab", Key: "cmd+shift+]", OnPick: func() { w.Step(1) }},
					{Title: "Previous Tab", Key: "cmd+shift+[", OnPick: func() { w.Step(-1) }},
				}},
				caution.Menu{Title: "Edit", Items: []caution.MenuItem{
					{Title: "Copy", Key: "cmd+c", OnPick: active(func(t *term.Term) { t.CopySelection() })},
					{Title: "Paste", Key: "cmd+v", OnPick: active((*term.Term).PasteClipboard)},
					{Sep: true},
					{Title: "Clear Scrollback", Key: "cmd+k", OnPick: active((*term.Term).ClearScrollback)},
				}},
				term.ViewMenu(w, func() { cfg.FontSize, cfg.Look = w.FontSize(), w.Look(); save(); menu() }),
			)
		}
		open()
		menu()
		return w.Node()
	}

	caution.ServeClient()
	if *addr != "" {
		log.Fatal(caution.Serve(*addr, mount))
	}
	err := native.Run(mount, native.Options{
		Title: "Caution Term", W: cfg.W, H: cfg.H, Maximized: cfg.Maximized,
		CustomTitlebar: true, TitlebarStyle: "compact",
		OnWindow: func(w, h int, maximized bool) {
			cfg.W, cfg.H, cfg.Maximized = w, h, maximized
			save()
		},
		Shot: *shot, SettleMs: *settle,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newWindow() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("new window: %v", err)
		return
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	if err := cmd.Start(); err != nil {
		log.Printf("new window: %v", err)
		return
	}
	go cmd.Wait()
}
